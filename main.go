package main

import (
	"bufio"
	// "expvar"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	logger    *log.Logger
	startTime time.Time
	allUsers  []User
	rooms     = make(map[string][]User)
	// To be implemented
	// roomsHistory    = make([][]string, 0, 10)
	categories    = make([]string, 0, 10)
	subChannel    = make(chan []interface{})
	newCatChannel = make(chan string)
	joinChan      = make(chan User)
	msgOutChan    = make(chan [2]string)
	//To be implemented
	// requestsMonitor = expvar.NewInt("Total Requests")
	// invalidRequests = expvar.NewInt("Total invalid requests")
	// totalUsers      = expvar.NewInt(("Total Users"))

)

type User struct {
	name    string
	address chan string
}

func init() {
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	logger = log.New(file, "", log.Ldate|log.Ltime|log.Lshortfile)

}

func startTCP() {
	listener, err := net.Listen("tcp", ":8000")
	if err != nil {
		log.Fatal(err)
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			logger.Println("user failed to connect")
		}

		go clientHandler(conn)

	}
}

func clientHandler(conn net.Conn) {

	scanner := bufio.NewScanner(conn)
	var newUser User
	logger.Printf("User %v connected\n", conn)
	io.WriteString(conn, "Welcome to the message hub!\n Write [CMD] at any time for a list of commands\n")
	io.WriteString(conn, "Server message: What is your name?\n")
	scanner.Scan()
	newUser.name = scanner.Text()
	newUser.address = make(chan string)

	joinChan <- newUser
	go func() {
		for {
			select {
			case newMsg := <-newUser.address:
				io.WriteString(conn, newMsg)
			}
		}
	}()
	go requestHandler(conn, newUser)
}

func userJoin() {
	for {
		select {
		case newUser := <-joinChan:
			entryMessage := fmt.Sprintf("%v has entered the server!\n", newUser.name)
			allUsers = append(allUsers, newUser)
			for _, v := range allUsers {
				v.address <- entryMessage
			}

		}
	}

}

func requestHandler(conn net.Conn, user User) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		switch fields[0] {
		case "CMD":
			io.WriteString(conn, "LIST: Shows list of categories\nPUB (category name): Publishes message in category subscribers\nSUB(category name): subscribes you to that category\n")
		case "LIST":
			io.WriteString(conn, "List of categories are:\n")
			for k := range rooms {
				io.WriteString(conn, fmt.Sprintf("%v\n", k))
			}

		case "SUB":
			var subArr = make([]interface{}, 2)
			subArr[0] = fields[1]
			subArr[1] = user
			subChannel <- subArr

		case "NEW":
			category := fields[1]
			newCatChannel <- category
			message := fmt.Sprintf("%v created a new channel, %v\n", user.name, category)
			for _, v := range allUsers {
				v.address <- message
			}

		case "PUB":
			room := fields[1]
			msg := fmt.Sprintf("%v wrote in %v channel: %v\n", user.name, room, strings.Join(fields[2:], " "))

			for _, v := range rooms[room] {
				v.address <- msg
			}
		}
	}
}

func main() {
	go startTCP()
	go userJoin()

	go func() {
		for {
			select {
			case newCat := <-newCatChannel:
				rooms[newCat] = make([]User, 0, 10)
			case newSub := <-subChannel:
				rooms[newSub[0].(string)] = append(rooms[newSub[0].(string)], newSub[1].(User))
				message := fmt.Sprintf("You have subscribed to channel %v\n", newSub[0].(string))
				newSub[1].(User).address <- message
			}
		}

	}()

	fmt.Scanln()

}
