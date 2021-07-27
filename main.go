package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sync"

	"golang.org/x/crypto/ssh"
)

func mux(w io.Writer, r io.Reader) (chan<- string, <-chan string) {
	in := make(chan string, 1)
	out := make(chan string, 1)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for cmd := range in {
			wg.Add(1)
			_, err := w.Write([]byte(cmd + "\n"))
			if err != nil {
				panic(err)
			}
			wg.Wait()
		}
	}()
	go func() {
		var (
			buf [65 * 1024]byte
			t   int
		)
		for {
			n, err := r.Read(buf[t:])
			if err != nil {
				close(in)
				close(out)
				return
			}
			t += n
			switch buf[t-1] {
			case '>', '#':
				out <- string(buf[:t])
				t = 0
				wg.Done()
			}
		}
	}()
	return in, out
}

func main() {
	var (
		endpoint = "192.168.1.1:22"
		username = "admin"
		password = "admin"
	)
	client, err := ssh.Dial("tcp", endpoint, &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	})
	if err != nil {
		log.Fatal("Failed to ssh.Dial(): ", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to ssh.NewSession(): ", err)
	}
	defer session.Close()

	w, err := session.StdinPipe()
	if err != nil {
		log.Fatal("Failed to session.StdinPipe(): ", err)
	}

	r, err := session.StdoutPipe()
	if err != nil {
		log.Fatal("Failed to session.StdoutPipe(): ", err)
	}

	in, out := mux(w, r)

	if err := session.Shell(); err != nil {
		log.Fatal("Failed to session.Shell(): ", err)
	}

	fmt.Printf(<-out)

	in <- "su"
	fmt.Printf(<-out)

	in <- "config"
	fmt.Printf(<-out)

	in <- "interface GigaEthernet 0/1"
	fmt.Printf(<-out)

	in <- "poe disable"
	fmt.Printf(<-out)

	in <- "show poe interface GigaEthernet 0/1"
	fmt.Printf(<-out)

	in <- "no poe disable"
	fmt.Printf(<-out)

	in <- "show poe interface GigaEthernet 0/1"
	fmt.Printf(<-out)

	in <- "exit"
	fmt.Printf(<-out)

	in <- "exit"
	fmt.Printf(<-out)

	in <- "exit"
	fmt.Printf(<-out)

	in <- "exit"
	fmt.Printf(<-out)

	if err := session.Wait(); err != nil {
		var errMissing *ssh.ExitMissingError
		if !errors.As(err, &errMissing) {
			log.Fatal("Failed to session.Wait(): ", err)
		}
	}
}
