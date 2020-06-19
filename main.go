package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/Gimulator/client-go"
	uuid "github.com/satori/go.uuid"
)

var (
	name string = "rando-agent" + uuid.NewV4().String()[0:5]
)

func main() {
	rand.Seed(time.Now().UnixNano())

	a, err := newAgent()
	if err != nil {
		panic(err)
	}
	a.listen()
}

type agent struct {
	*client.Client

	ch chan client.Object
}

func newAgent() (*agent, error) {
	ch := make(chan client.Object)

	cli, err := client.NewClient(ch)
	if err != nil {
		return nil, err
	}

	if err := cli.Set(client.Key{
		Namespace: "quarto",
		Name:      name,
		Type:      "register",
	}, ""); err != nil {
		return nil, err
	}

	if err := cli.Watch(client.Key{
		Namespace: "quarto",
		Name:      "board",
		Type:      "verdict",
	}); err != nil {
		return nil, err
	}

	return &agent{
		ch:     ch,
		Client: cli,
	}, nil
}

func (a *agent) listen() {
	for {
		fmt.Println("starting to listen")
		obj := <-a.ch
		fmt.Printf("starting to handle new object with key=%v and meta=%v", obj.Key, obj.Meta)

		board := Board{}
		err := json.Unmarshal([]byte(obj.Value), &board)
		if err != nil {
			fmt.Println("could not unmarshal data to board struct:", err.Error())
			continue
		}

		if board.Turn != name {
			fmt.Println("turn does not match with agent's name")
			continue
		}

		if err := a.action(board); err != nil {
			fmt.Println("could not execute action:", err.Error())
			continue
		}
	}
}

func (a *agent) action(board Board) error {
	avPieces := make([]int, 0)
	avPositions := make([]Position, 0)

	for id := range board.Pieces {
		isUsed := false
		for _, pos := range board.Positions {
			if pos.Piece == id {
				isUsed = true
			}
		}
		if !isUsed && id != board.Picked {
			avPieces = append(avPieces, id)
		}
	}

	for _, pos := range board.Positions {
		if pos.Piece == 0 {
			avPositions = append(avPositions, pos)
		}
	}

	if len(avPieces) == 0 {
		os.Exit(0)
	}

	n := rand.Intn(len(avPieces))

	ac := Action{
		Picked: avPieces[n],
		X:      avPositions[n].X,
		Y:      avPositions[n].Y,
	}

	b, err := json.Marshal(ac)
	if err != nil {
		return err
	}

	if err := a.Set(client.Key{
		Namespace: "quarto",
		Name:      name,
		Type:      "action",
	}, string(b)); err != nil {
		return err
	}

	return nil
}

//********************************* types
type Board struct {
	Pieces    map[int]Piece
	Positions []Position
	Turn      string
	Picked    int
}

type Position struct {
	X     int
	Y     int
	Piece int
}

type Piece struct {
	Length string
	Shape  string
	Color  string
	Hole   string
}

type Action struct {
	Picked int
	X      int
	Y      int
}
