package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	userName     = "[ENTER YOUR NAME]:"
	PORT         string
	entering     = make(chan message)
	leaving      = make(chan message)
	messages     = make(chan message)
	clients      = make(map[string]net.Conn)
	history      []string
	moveCursorUp = "\033[A"
	mutex        sync.Mutex
)

type client chan<- string

type message struct {
	address string
	text    string
	history []string
}

func broadcaster(mutex *sync.Mutex) {
	for {
		select {
		case msg := <-messages:
			mutex.Lock()
			for _, c := range clients {
				if msg.address == c.RemoteAddr().String() {
					fmt.Fprint(c, moveCursorUp+msg.text)
					continue
				} else {
					fmt.Fprint(c, msg.text)
				}
			}
			mutex.Unlock()
		case msg := <-entering:
			mutex.Lock()
			for _, c := range clients {
				if msg.address == c.RemoteAddr().String() {
					for _, w := range msg.history {
						fmt.Fprint(c, w+"\n")
					}
				} else {
					fmt.Fprint(c, msg.text)
				}
			}
			mutex.Unlock()
		case msg := <-leaving:
			mutex.Lock()
			for _, c := range clients {
				fmt.Fprint(c, msg.text)
			}
			mutex.Unlock()
		}
	}
}

func handleConnection(c net.Conn, mutex *sync.Mutex) {
	userName, err := makeNetData(c)
	if err != nil {
		return
	}

	if isLatinorCirillic(userName) {
		c.Write([]byte("[ENTER YOUR NAME]:"))
		handleConnection(c, mutex)
		return
	}

	if tenClients(c, nil) {
		return
	}

	user := "[" + userName + "]"

	if noSearchName(clients, userName) {
		mutex.Lock()
		clients[userName] = c
		mutex.Unlock()
	} else {
		fmt.Fprint(c, "User with the same name exists\n[ENTER YOUR NAME]:")
		handleConnection(c, mutex)
		return
	}
	mutex.Lock()
	entering <- newMessage(userName+" has joined our chat...", c)
	fmt.Println("clients", len(clients), clients)
	mutex.Unlock()

	input := bufio.NewScanner(c)
	for input.Scan() {
		myTime := time.Now().Format("2006-01-02 15:04:05")
		myTime = "[" + myTime + "]"
		if isLatinorCirillic(input.Text()) {
			continue
		}
		history = append(history, myTime+user+":"+input.Text())
		mutex.Lock()
		messages <- newMessage(myTime+user+":"+input.Text(), c)
		mutex.Unlock()

	}

	mutex.Lock()
	leaving <- newMessage(userName+" has left our chat...", c)
	c.Close()
	delete(clients, userName)
	fmt.Println("clients", len(clients), clients)
	mutex.Unlock()
}

func newMessage(msg string, c net.Conn) message {
	addr := c.RemoteAddr().String()
	return message{
		address: addr,
		text:    msg + "\n",
		history: history,
	}
}

func main() {
	arguments := os.Args
	if len(arguments) == 1 {
		PORT = ":8989"
	} else if len(arguments) == 2 {
		PORT = ":" + arguments[1]
	} else {
		fmt.Println("[USAGE]: ./TCPChat $port")
		return
	}

	l, err := net.Listen("tcp4", PORT)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer l.Close()

	penguin, err := makePenguin()
	if err != nil {
		return
	}

	fmt.Println("Listening on the port", PORT)
	go broadcaster(&mutex)
	for {
		c, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			return
		}
		mutex.Lock()
		tenClients(c, &mutex)
		mutex.Unlock()

		c.Write([]byte("Welcome to TCP-Chat!" + "\n" + penguin + "\n" + userName))

		go handleConnection(c, &mutex)
	}
}

func makeNetData(c net.Conn) (string, error) {
	netData, err := bufio.NewReader(c).ReadString('\n')
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	temp := strings.TrimSpace(string(netData))
	return temp, nil
}

func noSearchName(users map[string]net.Conn, user string) bool {
	if user == "" {
		return false
	}

	if len(users) == 0 {
		return true
	} else {
		for key := range users {
			if key == user {
				return false
			}
		}
	}

	return true
}

func tenClients(c net.Conn, mutex *sync.Mutex) bool {
	if len(clients) > 9 {

		c.Write([]byte("Sorry. Port full of users"))
		c.Close()
		return true
	}

	return false
}

func isLatinorCirillic(str string) bool {
	var num int
	for _, w := range str {
		if w == ' ' {
			num++
		}
	}

	if len(str) == num {
		return true
	}
	str = strings.TrimSuffix(str, "\n")
	rxmsg := regexp.MustCompile("^[\u0020-\u007f\u0400-\u04ff]+$")
	if rxmsg.MatchString(str) {
		return false
	}

	return true
}

func makePenguin() (string, error) {
	filetxt, err := os.ReadFile("penguin.txt")
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	return string(filetxt), nil
}
