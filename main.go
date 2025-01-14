package main

import (
	"flag"
	"log"

	"github.com/xddlg/algochat/chatstream"
	"github.com/xddlg/algochat/ui"
)

var (
	chatAddr       = flag.String("chataddr", "KPLD4GPZYXST7S2ALYSAVRCBWYBCUQCN6T4N6HAYCHCP4GOV7KWJUGITBE", "wallet address of chat server node")
	algodAddress   = flag.String("algodaddress", "http://localhost:8080", "algod.net address")
	algodToken     = flag.String("algodtoken", "", "algod.token value")
	kmdAddress     = flag.String("kmdaddress", "http://localhost:7833", "kmd.net address")
	kmdToken       = flag.String("kmdtoken", "", "kmd.token value")
	walletName     = flag.String("wallet", "", "the name of the wallet to use")
	walletPassword = flag.String("walletpassword", "", "the password of the wallet")
	fromAddr       = flag.String("from", "", "the addr of the wallet from which you will pay the txn fees")
	username       = flag.String("username", "Guest", "username to use in the chat")
)

func main() {
	flag.Parse()
	ams := chatstream.NewChatStream(*walletName, *walletPassword, *fromAddr, *username)

	if err := ams.Init(*chatAddr, *algodAddress, *algodToken, *kmdAddress, *kmdToken); err != nil {
		log.Fatalf("%v\n", err)
	}
	if err := ams.Run(); err != nil {
		log.Fatalf("%v\n", err)
	}

	in, out, logg := ams.GetInOut()
	if err := ui.StartAndLoop(in, out, logg); err != nil {
		log.Fatalf("%v\n", err)
	}
}
