package test

import (
	"redis-server/client"
	"redis-server/server"
	"testing"
	"time"
)

func startServer() {
	go func() {
		s := server.NewServer()
		s.Start(":6379")
	}()
	time.Sleep(time.Second) // Wait for server to start
}

func TestSetGet(t *testing.T) {
	startServer()

	c, err := client.NewClient("localhost:6379")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	// SET
	if err := c.Send("SET", "foo", "bar"); err != nil {
		t.Fatal(err)
	}

	reply, err := c.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if reply != "OK" {
		t.Errorf("Expected OK, got %s", reply)
	}

	// GET
	if err := c.Send("GET", "foo"); err != nil {
		t.Fatal(err)
	}

	reply, err = c.Receive()
	if err != nil {
		t.Fatal(err)
	}
	if reply != "bar" {
		t.Errorf("Expected bar, got %s", reply)
	}

	//EXISTS
	if err := c.Send("EXISTS", "foo"); err != nil {
		t.Fatal(err)
	}

	reply2, err := c.Receive()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(reply2)
	if reply2 != "1" {
		t.Errorf("Expected 1, got %s", reply2)
	}

	// //DEL
	// if err := c.Send("DEL", "foo"); err != nil {
	// 	t.Fatal(err)
	// }

	// reply1, err := c.Receive()
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// if reply1 != "1" {
	// 	t.Errorf("Expected 1, got %s", reply)
	// }

	// //EXISTS
	// if err := c.Send("EXISTS", "foo"); err != nil {
	// 	t.Fatal(err)
	// }

	// reply3, err := c.Receive()
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// // fmt.Println(reply2)
	// if reply3 != "0" {
	// 	t.Errorf("Expected 0, got %s", reply3)
	// }

	//EXPIRE
	if err := c.Send("EXPIRE", "foo", "10"); err != nil {
		t.Fatal(err)
	}

	reply4, err := c.Receive()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(reply2)
	if reply4 != "1" {
		t.Errorf("Expected 1, got %s", reply4)
	}

	//EXISTS
	if err := c.Send("EXISTS", "foo"); err != nil {
		t.Fatal(err)
	}

	reply3, err := c.Receive()
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(reply2)
	if reply3 != "1" {
		t.Errorf("Expected 1, got %s", reply3)
	}

	time.Sleep(time.Second * 10)

	//EXISTS
	if err := c.Send("TTL", "foo"); err != nil {
		t.Fatal(err)
	}

	reply5, err := c.Receive()
	if err != nil {
		t.Fatal(err)
	}

	t.Error(reply5)
	// if reply5 != "0" {
	// 	t.Errorf("Expected 0, got %s", reply3)
	// }
}
