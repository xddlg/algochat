package algochat

// ChatMessage is a chat message
type ChatMessage struct {
	Addr     		string
	Reputation	string
	Round				string
	Username 		string `json:username`
	Message  		string `json:message`
}
