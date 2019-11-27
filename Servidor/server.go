package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
	"strconv"
)

var cont = 1
var pai = 0

const (
	CONN_TYPE = "tcp"

	MAX_CLIENTS = 10

	CMD_PREFIX = "/"
	CMD_CREATE = CMD_PREFIX + "create"
	CMD_LIST   = CMD_PREFIX + "list"
	CMD_JOIN   = CMD_PREFIX + "join"
	CMD_LEAVE  = CMD_PREFIX + "leave"
	CMD_HELP   = CMD_PREFIX + "help"
	CMD_NAME   = CMD_PREFIX + "name"
	CMD_QUIT   = CMD_PREFIX + "quit"

	CLIENT_NAME = "Anonymous"
	SERVER_NAME = "Server"

	ERROR_PREFIX = "Error: "
	ERROR_SEND   = ERROR_PREFIX + "You cannot send messages in the lobby.\n"
	ERROR_CREATE = ERROR_PREFIX + "A chat room with that name already exists.\n"
	ERROR_JOIN   = ERROR_PREFIX + "A chat room with that name does not exist.\n"
	ERROR_LEAVE  = ERROR_PREFIX + "You cannot leave the lobby.\n"

	NOTICE_PREFIX          = "Notice: "
	NOTICE_ROOM_JOIN       = NOTICE_PREFIX + "\"%s\" joined the chat room.\n"
	NOTICE_ROOM_LEAVE      = NOTICE_PREFIX + "\"%s\" left the chat room.\n"
	NOTICE_ROOM_NAME       = NOTICE_PREFIX + "\"%s\" changed their name to \"%s\".\n"
	NOTICE_ROOM_DELETE     = NOTICE_PREFIX + "Chat room is inactive and being deleted.\n"
	NOTICE_PERSONAL_CREATE = NOTICE_PREFIX + "Created chat room \"%s\".\n"
	NOTICE_PERSONAL_NAME   = NOTICE_PREFIX + "Changed name to \"\".\n"

	MSG_CONNECT    = "Welcome to the server! Type \"/help\" to get a list of commands.\n"
	MSG_FULL       = "Server is full. Please try reconnecting later."
	MSG_DISCONNECT = "Disconnected from the server.\n"

	EXPIRY_TIME time.Duration = 7 * 24 * time.Hour
)


// Funcao que le o conteudo do arquivo e retorna um slice the string com todas as linhas do arquivo
func lerTexto(caminhoDoArquivo string) ([]string, error) {
	// Abre o arquivo
	arquivo, err := os.Open(caminhoDoArquivo)
	// Caso tenha encontrado algum erro ao tentar abrir o arquivo retorne o erro encontrado
	if err != nil {
		return nil, err
	}
	// Garante que o arquivo sera fechado apos o uso
	defer arquivo.Close()

	// Cria um scanner que le cada linha do arquivo
	var linhas []string
	scanner := bufio.NewScanner(arquivo)
	for scanner.Scan() {
		linhas = append(linhas, scanner.Text())
	}

	// Retorna as linhas lidas e um erro se ocorrer algum erro no scanner
	return linhas, scanner.Err()
}

// Funcao que escreve um texto no arquivo e retorna um erro caso tenha algum problema
func escreverTexto(linhas []string, caminhoDoArquivo string) error {
	// Cria o arquivo de texto
	arquivo, err := os.Create(caminhoDoArquivo)
	// Caso tenha encontrado algum erro retornar ele
	if err != nil {
		return err
	}
	// Garante que o arquivo sera fechado apos o uso
	defer arquivo.Close()

	// Cria um escritor responsavel por escrever cada linha do slice no arquivo de texto
	escritor := bufio.NewWriter(arquivo)
	for _, linha := range linhas {
		fmt.Fprintln(escritor, linha)
	}

	// Caso a funcao flush retorne um erro ele sera retornado aqui tambem
	return escritor.Flush()
}


// A Lobby receives messages on its channels, and keeps track of the currently
// connected clients, and currently created chat rooms.
type Server struct {
	name     string
	incoming chan *MessageServer
	outgoing chan string
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
}

func NewServer(conn net.Conn) *Server {
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	server := &Server{
		name:     CLIENT_NAME,
		incoming: make(chan *MessageServer),
		outgoing: make(chan string),
		conn:     conn,
		reader:   reader,
		writer:   writer,
	}

	server.Listen()
	return server
}
func (server *Server) Listen() {
	go server.Read()
	go server.Write()
}
func (server *Server) Read() {
	for {
		str, err := server.reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			break
		}
		temp := strings.Split(str, "#|#")
		var message *MessageServer
		if temp[0] == "05" {
			message = NewMessageServer(temp[0], temp[2], temp[1])
		} else {
			message = NewMessageServer(temp[0], "", temp[1])
		}
		server.incoming <- message
	}
	close(server.incoming)
	log.Println("Closed server's incoming channel read thread")
}

// Reads in messages from the Client's outgoing channel, and writes them to the
// Client's socket.
func (server *Server) Write() {
	for str := range server.outgoing {
		_, err := server.writer.WriteString(str)
		if err != nil {
			log.Println(err)
			break
		}
		err = server.writer.Flush()
		if err != nil {
			log.Println(err)
			break
		}
	}
	log.Println("Closed server's write thread")
}

type Lobby struct {
	ServerName            string
	clients               []*Client
	servers               []*Server
	chatRooms             map[string]*ChatRoom
	incoming              chan *Message
	join                  chan *Client
	leave                 chan *Client
	delete                chan *ChatRoom
	ServerConnect         string
	ServerListConnections map[string]string
	serverChanIn          chan *MessageServer
}

//NewLobby : Creates a lobby which beings listening over its channels.
func NewLobby(portx string) *Lobby {
	lobby := &Lobby{
		ServerName:   portx,
		clients:      make([]*Client, 0),
		chatRooms:    make(map[string]*ChatRoom),
		incoming:     make(chan *Message),
		join:         make(chan *Client),
		leave:        make(chan *Client),
		delete:       make(chan *ChatRoom),
		serverChanIn: make(chan *MessageServer),
	}
	go lobby.LobbyStart()
	lobby.Listen()
	return lobby
}

// LobbyStart config
func (lobby *Lobby) LobbyStart() {
	var wg sync.WaitGroup
	data, err := ioutil.ReadFile("serverconfig.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	//config made a map with all configuration from the server
	//that include all ports that we used from new connections and new listeners
	//imported from a Json file
	config := make(map[string]map[string]interface{})
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println(err)
		return
	}
	ServerListConnections := make(map[string]string)
	ServerConnect := config["ServerListenServer"]

	for i, k := range ServerConnect {
		if i == lobby.ServerName {
			continue
		}
		Temp := fmt.Sprintf("%v", k)
		ServerListConnections[Temp] = "NoConnected"
	}
	var portInit string
	for i, k := range ServerConnect {
		if lobby.ServerName == i {
			portInit = fmt.Sprintf("%v", k)
		}
	}

	//Listener Server conn
	Slistener, err := net.Listen(CONN_TYPE, portInit)
	if err != nil {
		log.Println("Error: ", err)
		os.Exit(1)
	}
	defer Slistener.Close()
	log.Println("Server Listening on " + portInit)

	wg.Add(3)
	go func() {
		for {
			Sconn, err := Slistener.Accept()
			if err != nil {
				log.Println("Error: ", err)
				continue

			}
			lobby.AddServer(Sconn)
			log.Printf("Deu certo carai!!!!!")
		}
	}()
	wg.Add(1)
	//fmt.Println(ServerList)
	// Function that attempts to connect to a new server
	go func() {
		for {
			for j, k := range ServerListConnections {
				fmt.Println("Server Port: " + j + " Status :" + k)
				if k == "Connected" {
					continue
				} else {
					if lobby.ServerConnectServer(j) == 1 {

						delete(ServerListConnections, j)
						ServerListConnections[j] = "Connected"
					}
				}
			}
			time.Sleep(10 * time.Second)
		}
	}()

	wg.Wait()

}

//AddServer kkk
func (lobby *Lobby) AddServer(newconn net.Conn) {
	server := NewServer(newconn)
	lobby.servers = append(lobby.servers, server)
	go func() {
		for message := range server.incoming {
			lobby.serverChanIn <- message
		}
	}()
}

//Listen Starts a new thread which listens over the Lobby's various channels.
func (lobby *Lobby) Listen() {
	go func() {
		for {
			select {
			case message := <-lobby.incoming:
				lobby.Parse(message)
			case client := <-lobby.join:
				lobby.Join(client)
			case client := <-lobby.leave:
				lobby.Leave(client)
			case chatRoom := <-lobby.delete:
				lobby.DeleteChatRoom(chatRoom)
			case server := <-lobby.serverChanIn:
				log.Print("funcaonova")
				log.Print(server)
			}
		}
	}()
}

//Join : Handles clients connecting to the lobby
func (lobby *Lobby) Join(client *Client) {
	if len(lobby.clients) >= MAX_CLIENTS {
		client.Quit()
		return
	}

	lobby.clients = append(lobby.clients, client)
	client.outgoing <- MSG_CONNECT
	go func() {
		for message := range client.incoming {
			lobby.incoming <- message
		}
		lobby.leave <- client
	}()
}

//Leave : Handles clients disconnecting from the lobby.
func (lobby *Lobby) Leave(client *Client) {
	if client.chatRoom != nil {
		client.chatRoom.Leave(client)
	}
	for i, otherClient := range lobby.clients {
		if client == otherClient {
			lobby.clients = append(lobby.clients[:i], lobby.clients[i+1:]...)
			break
		}
	}
	close(client.outgoing)
	log.Println("Closed client's outgoing channel")
}

//DeleteChatRoom Checks if the a channel has expired. If it has, the chat room is deleted.
// Otherwise, a signal is sent to the delete channel at its new expiry time.
func (lobby *Lobby) DeleteChatRoom(chatRoom *ChatRoom) {
	if chatRoom.expiry.After(time.Now()) {
		go func() {
			time.Sleep(chatRoom.expiry.Sub(time.Now()))
			lobby.delete <- chatRoom
		}()
		log.Println("attempted to delete chat room")
	} else {
		chatRoom.Delete()
		delete(lobby.chatRooms, chatRoom.name)
		log.Println("deleted chat room")
	}
}

//Parse : Handles messages sent to the lobby. If the message contains a command, the
// command is executed by the lobby. Otherwise, the message is sent to the
// sender's current chat room.
func (lobby *Lobby) Parse(message *Message) {
	switch {
	default:
		lobby.SendMessage(message)
	case strings.HasPrefix(message.text, CMD_CREATE):
		name := strings.TrimSuffix(strings.TrimPrefix(message.text, CMD_CREATE+" "), "\n")
		lobby.CreateChatRoom(message.client, name)
	case strings.HasPrefix(message.text, CMD_LIST):
		lobby.ListChatRooms(message.client)
	case strings.HasPrefix(message.text, CMD_JOIN):
		name := strings.TrimSuffix(strings.TrimPrefix(message.text, CMD_JOIN+" "), "\n")
		lobby.JoinChatRoom(message.client, name)
	case strings.HasPrefix(message.text, CMD_LEAVE):
		lobby.LeaveChatRoom(message.client)
	case strings.HasPrefix(message.text, CMD_NAME):
		name := strings.TrimSuffix(strings.TrimPrefix(message.text, CMD_NAME+" "), "\n")
		lobby.ChangeName(message.client, name)
	case strings.HasPrefix(message.text, CMD_HELP):
		lobby.Help(message.client)
	case strings.HasPrefix(message.text, CMD_QUIT):
		message.client.Quit()
	}
}

//SendMessage Attempts to send the given message to the client's current chat room. If they
// are not in a chat room, an error message is sent to the client.
func (lobby *Lobby) SendMessage(message *Message) {
	if message.client.chatRoom == nil {
		message.client.outgoing <- ERROR_SEND
		log.Println("client tried to send message in lobby")
		return
	}
	message.client.chatRoom.Broadcast(message.String(), lobby)
	log.Println("client " + message.client.name + " sent message " + message.text + " to the room " + message.client.chatRoom.name + " .")
}

//CreateChatRoom Attempts to create a chat room with the given name, provided that one does
// not already exist.
func (lobby *Lobby) CreateChatRoom(client *Client, name string) {
	if lobby.chatRooms[name] != nil {
		client.outgoing <- ERROR_CREATE
		log.Println("client tried to create chat room with a name already in use")
		return
	}
	chatRoom := NewChatRoom(name)
	lobby.chatRooms[name] = chatRoom
	go func() {
		time.Sleep(EXPIRY_TIME)
		lobby.delete <- chatRoom
	}()
	client.outgoing <- fmt.Sprintf(NOTICE_PERSONAL_CREATE, chatRoom.name)
	log.Println("client created chat room")
}

//JoinChatRoom : Attempts to add the client to the chat room with the given name, provided
// that the chat room exists.
func (lobby *Lobby) JoinChatRoom(client *Client, name string) {
	if lobby.chatRooms[name] == nil {
		client.outgoing <- ERROR_JOIN
		log.Println("client tried to join a chat room that does not exist")
		return
	}
	if client.chatRoom != nil {
		lobby.LeaveChatRoom(client)
	}
	lobby.chatRooms[name].Join(client)
	log.Println("client joined chat room")
}

//LeaveChatRoom : Removes the given client from their current chat room.
func (lobby *Lobby) LeaveChatRoom(client *Client) {
	if client.chatRoom == nil {
		client.outgoing <- ERROR_LEAVE
		log.Println("client tried to leave the lobby")
		return
	}
	client.chatRoom.Leave(client)
	log.Println("client left chat room")
}

//ChangeName : Changes the client's name to the given name.
func (lobby *Lobby) ChangeName(client *Client, name string) {
	if client.chatRoom == nil {
		client.outgoing <- fmt.Sprintf(NOTICE_PERSONAL_NAME, name)
	} else {
		client.chatRoom.Broadcast(fmt.Sprintf(NOTICE_ROOM_NAME, client.name, name), nil)
	}
	client.name = name
	log.Println("client changed their name")
}

//ListChatRooms : Sends to the client the list of chat rooms currently open.
func (lobby *Lobby) ListChatRooms(client *Client) {
	client.outgoing <- "\n"
	client.outgoing <- "Chat Rooms:\n"
	for name := range lobby.chatRooms {
		client.outgoing <- fmt.Sprintf("%s\n", name)
	}
	client.outgoing <- "\n"
	log.Println("client listed chat rooms")
}

//Help : Sends to the client the list of possible commands to the client.
func (lobby *Lobby) Help(client *Client) {
	client.outgoing <- "\n"
	client.outgoing <- "Commands:\n"
	client.outgoing <- "/help - lists all commands\n"
	client.outgoing <- "/list - lists all chat rooms\n"
	client.outgoing <- "/create foo - creates a chat room named foo\n"
	client.outgoing <- "/join foo - joins a chat room named foo\n"
	client.outgoing <- "/leave - leaves the current chat room\n"
	client.outgoing <- "/name foo - changes your name to foo\n"
	client.outgoing <- "/quit - quits the program\n"
	client.outgoing <- "\n"
	log.Println("client requested help")
}

// A ChatRoom contains the chat's name, a list of the currently connected
// clients, a history of the messages broadcast to the users in the channel,
// and the current time at which the ChatRoom will expire.
type ChatRoom struct {
	name     string
	clients  []*Client
	messages []string
	expiry   time.Time
}

//NewChatRoom : Creates an empty chat room with the given name, and sets its expiry time to
// the current time + EXPIRY_TIME.
func NewChatRoom(name string) *ChatRoom {
	return &ChatRoom{
		name:     name,
		clients:  make([]*Client, 0),
		messages: make([]string, 0),
		expiry:   time.Now().Add(EXPIRY_TIME),
	}
}

//Join  Adds the given Client to the ChatRoom, and sends them all messages that have
// that have been sent since the creation of the ChatRoom.
func (chatRoom *ChatRoom) Join(client *Client) {
	client.chatRoom = chatRoom
	for _, message := range chatRoom.messages {
		client.outgoing <- message
	}
	chatRoom.clients = append(chatRoom.clients, client)
	chatRoom.Broadcast(fmt.Sprintf(NOTICE_ROOM_JOIN, client.name), nil)
}

//Leave : Removes the given Client from the ChatRoom.
func (chatRoom *ChatRoom) Leave(client *Client) {
	chatRoom.Broadcast(fmt.Sprintf(NOTICE_ROOM_LEAVE, client.name), nil)
	for i, otherClient := range chatRoom.clients {
		if client == otherClient {
			chatRoom.clients = append(chatRoom.clients[:i], chatRoom.clients[i+1:]...)
			break
		}
	}
	client.chatRoom = nil
}

//Broadcast : Sends the given message to all Clients currently in the ChatRoom.
func (chatRoom *ChatRoom) Broadcast(message string, lobby *Lobby) {
	chatRoom.expiry = time.Now().Add(EXPIRY_TIME)
	chatRoom.messages = append(chatRoom.messages, message)
	for _, client := range chatRoom.clients {
		client.outgoing <- message
	}
	if lobby != nil {
		Smessage := NewMessageServer("05", message, chatRoom.name)
		for _, server := range lobby.servers {
			server.outgoing <- Smessage.String()
		}
	}
}

//Delete : Notifies the clients within the chat room that it is being deleted, and kicks
// them back into the lobby.
func (chatRoom *ChatRoom) Delete() {
	//notify of deletion?
	chatRoom.Broadcast(NOTICE_ROOM_DELETE, nil)
	for _, client := range chatRoom.clients {
		client.chatRoom = nil
	}
}

// A Client abstracts away the idea of a connection into incoming and outgoing
// channels, and stores some information about the client's state, including
// their current name and chat room.
type Client struct {
	name     string
	chatRoom *ChatRoom
	incoming chan *Message
	outgoing chan string
	conn     net.Conn
	reader   *bufio.Reader
	writer   *bufio.Writer
}

// NewClient :Returns a new client from the given connection, and starts a reader and
// writer which receive and send information from the socket
func NewClient(conn net.Conn) *Client {
	writer := bufio.NewWriter(conn)
	reader := bufio.NewReader(conn)

	client := &Client{
		name:     CLIENT_NAME,
		chatRoom: nil,
		incoming: make(chan *Message),
		outgoing: make(chan string),
		conn:     conn,
		reader:   reader,
		writer:   writer,
	}

	client.Listen()
	return client
}

// Listen : Starts two threads which read from the client's outgoing channel and write to
// the client's socket connection, and read from the client's socket and write
// to the client's incoming channel.
func (client *Client) Listen() {
	go client.Read()
	go client.Write()
}

// Reads in strings from the Client's socket, formats them into Messages, and
// puts them into the Client's incoming channel.
func (client *Client) Read() {
	for {
		str, err := client.reader.ReadString('\n')
		if err != nil {
			log.Println(err)
			break
		}
		message := NewMessage(time.Now(), client, strings.TrimSuffix(str, "\n"))
		client.incoming <- message
	}
	close(client.incoming)
	log.Println("Closed client's incoming channel read thread")
}

// Reads in messages from the Client's outgoing channel, and writes them to the
// Client's socket.
func (client *Client) Write() {
	for str := range client.outgoing {
		_, err := client.writer.WriteString(str)
		if err != nil {
			log.Println(err)
			break
		}
		err = client.writer.Flush()
		if err != nil {
			log.Println(err)
			break
		}
	}
	log.Println("Closed client's write thread")
}

// Quit : Closes the client's connection. Socket closing is by error checking, so this
// takes advantage of that to simplify the code and make sure all the threads
// are cleaned up.
func (client *Client) Quit() {
	client.conn.Close()
}

// A Message contains information about the sender, the time at which the
// message was sent, and the text of the message. This gives a convenient way
// of passing the necessary information about a message from the client to the
// lobby.
type Message struct {
	time   time.Time
	client *Client
	text   string
}

// NewMessage : Creates a new message with the given time, client and text.
func NewMessage(time time.Time, client *Client, text string) *Message {
	return &Message{
		time:   time,
		client: client,
		text:   text,
	}
}

// Returns a string representation of the message.
func (message *Message) String() string {

	var conteudo []string
	conteudo, err := lerTexto("logMsg.csv")

	var cont2 = strconv.Itoa(cont)
	var pai2 = strconv.Itoa(pai)
	var hora = time.Now()
	var sala = message.client.chatRoom.name
	var cliente = message.client.name
	var texto = message.text

	conteudo = append(conteudo, cont2 + ";" + pai2 + ";" + hora.Format("3:04PM") + ";" + sala + ";" + cliente + ";" + texto)

	cont = cont + 1
	pai = pai + 1

	err = escreverTexto(conteudo, "logMsg.csv")
	if err != nil {
		log.Fatalf("Erro:", err)
	}

	return fmt.Sprintf("%s - %s: %s\n", message.time.Format(time.Kitchen), message.client.name, message.text)
}

//MessageServer
//prefix 00-> NewRoom,Delete,Join
type MessageServer struct {
	prefix  string
	info    string
	message string
}

//NewMessageServer
func NewMessageServer(code string, message1 string, info1 string) *MessageServer {
	return &MessageServer{
		prefix:  code,
		message: message1,
		info:    info1,
	}
}
func (messageserver *MessageServer) String() string {
	if messageserver.prefix == "00" {
		//prefix 00 - New Room
		return fmt.Sprintf("%s#|#%s#|#", messageserver.prefix, messageserver.info)
	} else if messageserver.prefix == "01" {
		//prefix 01 - Join
		return fmt.Sprintf("%s#|#%s#|#", messageserver.prefix, messageserver.info)
	} else if messageserver.prefix == "02" {
		//prefix 02 - Leave
		return fmt.Sprintf("%s#|#%s#|#", messageserver.prefix, messageserver.info)
	} else if messageserver.prefix == "03" {
		//prefix 03 - Delete
		return fmt.Sprintf("%s#|#%s#|#", messageserver.prefix, messageserver.info)
	}
	return fmt.Sprintf("%s#|#%s#|#%s\n", messageserver.prefix, messageserver.info, messageserver.message)
}

//Server functions
//ServerRead
func ServerRead(conn net.Conn) {
	reader := bufio.NewReader(conn)
	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf(MSG_DISCONNECT)
			return
		}
		fmt.Print(str)
	}
}

func ServerWrite(conn net.Conn) {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(conn)

	for {
		str, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		_, err = writer.WriteString(str)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		err = writer.Flush()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
	}
}

// ServerConnectServer return 1 for acc connections or 0 for connections refused
func (lobby *Lobby) ServerConnectServer(ConnPort string) int {

	conn, err := net.Dial(CONN_TYPE, ConnPort)
	if err != nil {
		//fmt.Println(err)
		return 0
	} else {

		go ServerRead(conn)
		go ServerWrite(conn)
		return 1
	}
}

// Creates a lobby, listens for client connections, and connects them to the
// lobby.

func main() {

	var wg sync.WaitGroup
	data, err := ioutil.ReadFile("serverconfig.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	//config made a map with all configuration from the server
	//that include all ports that we used from new connections and new listeners
	//imported from a Json file
	config := make(map[string]map[string]interface{})
	err = json.Unmarshal(data, &config)
	if err != nil {
		fmt.Println(err)
		return
	}
	//ServerList := make(map[string]string)
	ServerPortInit := config["ServerPortInit"]
	ServerConnections := config["ServerListenServer"]

	//logs
	//*
	fmt.Println(ServerConnections)
	fmt.Println(ServerPortInit)
	fmt.Println(os.Args[1])
	fmt.Println(config)
	ConnPort := fmt.Sprintf("%v", ServerPortInit[os.Args[1]])
	//ConnServerPort := fmt.Sprintf("%v", ServerConnections[os.Args[1]])
	fmt.Println(ConnPort)
	//*/

	log.SetFlags(log.LstdFlags | log.Lshortfile)

	lobby := NewLobby(os.Args[1])

	//Listener clients conn
	listener, err := net.Listen(CONN_TYPE, ConnPort)
	if err != nil {
		log.Println("Error: ", err)
		os.Exit(1)
	}
	defer listener.Close()
	log.Println("Listening on " + ConnPort)

	// New client conn func
	wg.Add(3)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Error: ", err)
				continue

			}
			lobby.Join(NewClient(conn))
		}
	}()
	wg.Wait()

}

// ata
