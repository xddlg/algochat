package chatstream

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"strconv"
	"sync"
	"time"

	"github.com/algorand/go-algorand-sdk/client/algod"
	"github.com/algorand/go-algorand-sdk/client/kmd"
	"github.com/algorand/go-algorand-sdk/transaction"
	"github.com/xddlg/algochat/algochat"
	"github.com/pkg/errors"
)

const (
	// Initial intention:
	// avgSecPerBlock     = 6 // approx
	// blocksPerDay       = 60 * 60 * 24 / avgSecPerBlock
	// statingDaysBefore  = 1
	// initialBlockOffset = blocksPerDay * statingDaysBefore

	// Max allowed nowadays:
	initialBlockOffset = 1000
	outChanCapacity    = 10
)

// AlgoChatStream receives new messages from the blockchain
type AlgoChatStream struct {
	in             chan *algochat.ChatMessage
	out            chan string
	logg           chan string
	fromAddr       string
	walletName     string
	walletPassword string
	walletID       string
	username       string

	chatAddr       string

	mux         sync.Mutex // guards everything below
	inited      bool
	running     bool
	algodClient algod.Client
	kmdClient   kmd.Client
}

// NewChatStream creates a stream of new messages
func NewChatStream(walletName, walletPassword, fromAddr, username string) *AlgoChatStream {
	ams := &AlgoChatStream{
		in:             make(chan *algochat.ChatMessage),
		out:            make(chan string, outChanCapacity),
		logg:           make(chan string, 10),
		username:       username,
		walletName:     walletName,
		walletPassword: walletPassword,
		fromAddr:       fromAddr,
	}
	return ams
}

// Init creates the algod and kmd client for interaction with node chatAddr
func (ams *AlgoChatStream) Init(chatAddr, algodAddress, algodToken, kmdAddress, kmdToken string) error {
	ams.mux.Lock()
	defer ams.mux.Unlock()
	if ams.inited {
		return nil
	}

	ams.chatAddr = chatAddr

	algodClient, err := algod.MakeClient(algodAddress, algodToken)
	if err != nil {
		return errors.Wrap(err, "failed to make algod client")
	}
	ams.algodClient = algodClient

	kmdClient, err := kmd.MakeClient(kmdAddress, kmdToken)
	if err != nil {
		return errors.Wrap(err, "failed to make kmd client")
	}
	ams.kmdClient = kmdClient
	wallets, err := kmdClient.ListWallets()
	if err != nil {
		return errors.Wrap(err, "couldn't list wallets")
	}
	for _, w := range wallets.Wallets {
		if strings.Compare(w.Name, ams.walletName) == 0 {
			ams.walletID = w.ID
			break
		}
	}
	if len(ams.walletID) == 0 {
		return errors.New("didn't find the wallet by its name")
	}

	ams.inited = true

	return nil
}

// GetInOut returns the channel where new messages will appear
func (ams *AlgoChatStream) GetInOut() (<-chan *algochat.ChatMessage, chan<- string, <-chan string) {
	return ams.in, ams.out, ams.logg
}

// Run starts inspecting new blocks for messages
func (ams *AlgoChatStream) Run() error {
	ams.mux.Lock()
	defer ams.mux.Unlock()
	if !ams.inited {
		return errors.New("not inited, you should init")
	}

	if !ams.running {
		ams.running = true
		go ams.listenNewMessages()
		go ams.sendMessages()
	}

	return nil
}

func (ams *AlgoChatStream) sendMessages() {
	for {
		select {
		case msg := <-ams.out:
			err := ams.sendMessagesInTrx(msg)
			if err != nil {
				log.Printf("%v\n", err)
				continue // could definitely be better ¯\_(ツ)_/¯
			}
		}
	}
}

func (ams *AlgoChatStream) sendMessagesInTrx(msg string) error {
	txParams, err := ams.algodClient.SuggestedParams()
	if err != nil {
		return errors.Wrap(err, "error getting suggested params")
	}
	msgBytes, err := json.Marshal(algochat.ChatMessage{Addr: ams.fromAddr, Message: msg, Username: ams.username})
	if err != nil {
		return errors.Wrap(err, "couldn't marshal msg")
	}

	tx, err := transaction.MakePaymentTxn(ams.fromAddr, ams.chatAddr, txParams.Fee, 0, txParams.LastRound, txParams.LastRound+100, msgBytes, "", txParams.GenesisID, txParams.GenesisHash)
	if err != nil {
		return errors.Wrap(err, "error creating the transaction")
	}

	iw, err := ams.kmdClient.InitWalletHandle(ams.walletID, ams.walletPassword)
	if err != nil {
		return errors.Wrap(err, "couldn't init the wallet")
	}
	signResponse, err := ams.kmdClient.SignTransaction(iw.WalletHandleToken, ams.walletPassword, tx)
	if err != nil {
		return errors.Wrap(err, "couldn't sign the transaction")
	}

	ams.logg <- fmt.Sprintf("Sending with fee %v microalgos...", tx.Fee)
	sentTxID, err := ams.algodClient.SendRawTransaction(signResponse.SignedTransaction)
	if err != nil {
		ams.logg <- "Failed to send the message!"
		return errors.Wrap(err, "failed sending the transaction")
	}
	ams.logg <- "Waiting for confirmation..."

	unconfirmedTx := true
	errorTx := ""
	for unconfirmedTx {
		time.Sleep(time.Millisecond * 100)
		txn, err := ams.algodClient.PendingTransactionInformation(sentTxID.TxID)
		if err != nil {
			errorTx = "Error querying pending transaction!"
			break
		}
		unconfirmedTx = txn.ConfirmedRound > 0
		errorTx = txn.PoolError
	}
	if errorTx == "" {
		ams.logg <- "Done, will appear soon!"
	} else {
		ams.logg <- "Error!: " + errorTx
	}

	return nil
}

func (ams *AlgoChatStream) listenNewMessages() {
	status, err := ams.algodClient.Status()

	if err != nil {
		panic("error while getting node status")
	}

	var blockNum uint64 = 1
	// the last 1000 (should it be less for us?)
	if(status.LastRound > initialBlockOffset) {
		blockNum = status.LastRound - initialBlockOffset
	}

	for {
		_, err := ams.algodClient.StatusAfterBlock(blockNum)
		if err != nil {
			log.Printf("%v\n", err)
			continue
		}

		b, err := ams.algodClient.Block(blockNum)

		if err != nil {
			blockNum++
			continue
		}

		for _, t := range b.Transactions.Transactions {
			if ( (t.Payment != nil) && (strings.Compare(t.Payment.To, ams.chatAddr) == 0) ) {
				message := &algochat.ChatMessage{}
				err = json.Unmarshal(t.Note, message)
				if err != nil {
					continue
				}
				message.Addr = t.From[:5]
				message.Round = strconv.FormatUint(blockNum, 10)
				ai, err := ams.algodClient.AccountInformation(t.From)
				if (err != nil) {
					message.Reputation = "N/A"
				} else {
					message.Reputation = fmt.Sprintf("%v", ai.Reputation)
				}

				ams.in <- message
			}
		}
		blockNum = blockNum + 1
	}
}
