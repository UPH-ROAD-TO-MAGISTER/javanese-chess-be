package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"javanese-chess/game9x9"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	g := game9x9.NewGame(
		[]struct {
			Name, Color string
			IsBot       bool
		}{
			{"You", "green", false},
			{"CPU", "red", true},
		},
		time.Now().UnixNano(),
	)
	g.Started = true

	reader := bufio.NewReader(os.Stdin)
	for !g.Finished {
		player := g.Players[g.TurnIdx]
		fmt.Printf("\nGiliran: %s (%s)\n", player.Name, player.Color)
		fmt.Printf("Hand: %v\n", player.Hand)
		fmt.Printf("Board:\n")
		printBoard(g)

		if player.IsBot {
			mv, ok := g.BotChooseMove(g.TurnIdx)
			if !ok {
				fmt.Println("Bot skip turn.")
				g.ApplyMove(game9x9.Move{}) // skip
				continue
			}
			fmt.Printf("Bot memilih: (%d,%d) kartu %d\n", mv.R+1, mv.C+1, mv.Card)
			_ = g.ApplyMove(mv)
		} else {
			moves := g.LegalMoves(g.TurnIdx)
			if len(moves) == 0 {
				fmt.Println("Tidak ada langkah legal, skip turn.")
				g.ApplyMove(game9x9.Move{}) // skip
				continue
			}
			fmt.Println("Masukkan langkah: format baris kolom kartu (contoh: 5 5 7)")
			for {
				fmt.Print("> ")
				line, _ := reader.ReadString('\n')
				parts := strings.Fields(line)
				if len(parts) != 3 {
					fmt.Println("Format salah. Coba lagi.")
					continue
				}
				r, _ := strconv.Atoi(parts[0])
				c, _ := strconv.Atoi(parts[1])
				card, _ := strconv.Atoi(parts[2])
				mv := game9x9.Move{PlayerID: player.ID, R: r - 1, C: c - 1, Card: card}
				err := g.ApplyMove(mv)
				if err != nil {
					fmt.Println("Langkah tidak valid:", err)
					continue
				}
				break
			}
		}
	}

	fmt.Println("\nPermainan selesai!")
	js, _ := json.MarshalIndent(g.Export(), "", "  ")
	fmt.Println(string(js))
}

func printBoard(g *game9x9.Game) {
	for r := 0; r < 9; r++ {
		for c := 0; c < 9; c++ {
			cell := g.Board[r][c]
			if cell.Owner == -1 {
				fmt.Print(". ")
			} else {
				fmt.Printf("%d ", cell.Value)
			}
		}
		fmt.Println()
	}
}
