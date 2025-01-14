# Algochat 

Algochat is an app that uses Algorand blockchain to implement a public chat service. 

It's not an official recommendation about making dapps mounted on Algorand, just a funny project to try the SDK and play around with the _note_ field which can hold arbitrary byte data. Dirty, non-production ready, only a PoC!

# Use
At the current moment, it works in the TestNet if no chat node wallet address is provided, so if you have an account with Algos you're ready to go.

```
$ go run main.go -h
Usage of main:
  -chataddr string
        the addr of the online account of the default wallet of node that serves as chat node (default "KPLD4GPZYXST7S2ALYSAVRCBWYBCUQCN6T4N6HAYCHCP4GOV7KWJUGITBE" on the TestNet)
  -algodaddress string
        algod.net address of chat client node (default "http://localhost:8080")
  -algodtoken string
        algod.token value of chat client node
  -from string
        the addr of the online account of the default wallet of the client node (the node from which you will pay the txn fees)
  -kmdaddress string
        kmd.net address of chat client node (default "http://localhost:7833")
  -kmdtoken string
        kmd.token value of chat client node
  -username string
        username to use in the chat (default "Guest")
  -wallet string
        the name of the wallet on chat client node to use
  -walletpassword string
        the password of the wallet on chat client node to use
```

Usage:
```
$ go run main.go -chataddr <chat-node-account-addr> -algodaddress <client-node-algod-net-address> -algodtoken <client-node-algod-token> -kmdaddress <client-node-kmd-net-address> -kmdtoken <client-node-kmd-token> -wallet <client-node-wallet-name> -walletpassword <client-node-wallet-password> -from <client-node-account-addr> -username <client-node-username>

```

You'll see all the chat messages in the last 1000 blocks, and you can immediately start chatting.

Here a 20s demo:
![image](demo.gif)

Check the logs to see what's happening.

## Instructions
### Prerequisites
1. Install golang, ```sudo apt-get install golang```
2. Create ```go``` folder and set environment path variables
``` 
cat >> ~/.bashrc << 'EOF'
export GOPATH=$HOME/go
export PATH=/usr/local/go/bin:$PATH:$GOPATH/bin
EOF 
```
```source ~/.bashrc```

3.  Verify you have the GOPATH env correctly: ```echo $GOPATH```
4. Install git, ```sudo apt-get install git```
### Installation
5. Inside the ```go``` folder, do: ```go get github.com/xddlg/algochat```
6. Navigate inside the folder above, i.e. ```cd $GOPATH/src/github.com/xddlg/algochat```

### Running the App on Private Local Network
7. Set up a private local network with at least two nodes. One as the chat node and another as the chat client node. Follow https://developer.algorand.org/docs/creating-private-network instructions all the way to _Creating a New Account and a Participation Key_. Identify the node that will be used as the chat node and the one that will be the chat client node. Keep wallet addresses, names, passwords and node tokens handy.
8. Start your network, which starts the nodes
>goal network start -r _path-to-your-network_
9. Start kmd for the chat client node
>goal kmd start -d _path-to-your-network_/_chat-client-node_
10. Get the network addresses from _path-to-your-network_/_chat-client-node_
11. Start the algochat for the chat client node. See usage, above.
12. When done, stop kmd and the network.
>goal network stop -r _path-to-your-network_

>goal kmd stop -d _path-to-your-network_/_chat-client-node_

### Running the App on TestNet
13. Make sure you start ```kmd``` as it's not started by default on the node (`goal kmd start`)
14. If using default config values, launch the AlgoChat app
>go run main.go -wallet name-of-your-wallet -from your-account-address -algodaddress http://192.168.1.1:8080 -algodtoken your-algod-token -kmdtoken your-kmd-token -kmdaddress http://192.168.1.1:7833 -username Guest
15. If using Windows and have an SSH client (e.g. Putty, SecureCRT), to display properly AlgoChat, you need in the appearance settings of the client to set character encoding to ```UTF-8```

Thanks to [nikandriko](https://github.com/nikandriko) for elaborating the above step-by-step instructions.

In case you're having troubles with the `algod.token` or `kmd.token`, stop both the node and kmd, delete the `.token` files, and start them again `goal node start`  (remember `-d` flag if you're not using the env variable config).




## Random notes
* If testing on a private network with more than one chat client node, it is more convenient to change the nodes config to have algod and kmd addresses to be in predifined ports, making it much easier to scritp the start / stop and having a web app talking to it.
* Whenever a user sends a message to the chatroom it gets marshaled in the _note_ field of the transaction. 
* Every transaction with the destination address _KPLD4GPZYXST7S2ALYSAVRCBWYBCUQCN6T4N6HAYCHCP4GOV7KWJUGITBE_ is considered a message of the app.
* Every message also shows the first 5 sringified characters of the address that sent the message as a prefix of the username.
* The message is JSON encoded. Considering the _note_ field size limitation, a more efficient encoding like protobufs would be more convenient.
* The transaction corresponding to each message sends 0 algos to the chat address and uses the suggested fee from the SDK.
* When you open the chat it will show the messages contained in the last 1000 blocks. Originally I wanted this to be parametrizable, but since the default node is non-archival it doesn't make much sense. 1000  blocks ~ 1-hour approx.
* The SDK allows sending the transaction only asynchronously. After the transaction is submitted, the program scans the pending transactions until the sent transactionID is out of the pool, then considers the transaction confirmed (... not 100% true).
* Ctrl+C will close the program immediately. The code doesn't worry about finishing goroutines gracefully.
* Everything you see in the _Algorand Chat_ window is exclusively confirmed on-chain. I intentionally avoided _cheating_ showing new messages as soon as is submitted (before real confirmation). In a production chat service we'd intentionally do the opposite, display it right away with some UX signals (async).

## Seems to be a good idea
* Event notifications _natively_ from the SDK, e.g, a new block has arrived.
* Synchronous transaction sending or confirmation callbacks.
* The SDK is a wrapper of the node REST API. Consider having gRPC endpoint on the node, might allow much powerful APIs? (use of streams for events, faster calls, less mem, etc).
