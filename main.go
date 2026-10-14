package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/vince/vinDB/btree"
	"github.com/vince/vinDB/encoding"
	"github.com/vince/vinDB/storage"
)

// CLI REPL for vinDB, needs to be called with a valid DB file
// Current supported commands:
// - insert <id> <user> <email>
// - select <fn (Row) bool>
// - .exit
func main() {
	args := os.Args
	if len(args) < 2 {
		fmt.Println("Usage: vinDB <database-file.db>")
    	return
	}
	parts := strings.Split(args[1], ".")

	if parts[len(parts)-1] != "db" {
		fmt.Printf("Invalid db File")
		return
	}

	pg, err := storage.Open(args[1])
	if err != nil {
		print(err)
	}
	defer pg.Close()

	bt, err := btree.NewBTree(pg,0)
	if err != nil {
		print(err)
	}
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("vinDB > ")
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.TrimSpace(line)
		if line == ".exit" {
			break
		}

		parts := strings.Split(line, " ")
		cmd := parts[0]
		switch cmd {
			case "insert":
				if len(parts) != 4 {
					fmt.Println("Usage: insert <id> <user> <email>")
					continue
				}

				id, err :=  strconv.Atoi(parts[1])
				if err != nil {
					fmt.Printf("ATOI Error: %v\n", err)
					continue
				}
				user := parts[2]
				email := parts[3]
				row := encoding.Row{
					ID: uint32(id),
					Username: user,
					Email: email,
				}
				rowBytes, err := row.SerializeRow()
				if err != nil {
					fmt.Print(err)
				}
				
				if err = bt.Insert(row.ID, rowBytes);err != nil {
					fmt.Printf("Insert Error: %v\n", err)
				}
			case "find":
				if len(parts) != 2 {
					fmt.Println("Usage: find <id>")
					continue
				}
				id, err :=  strconv.Atoi(parts[1])
				if err != nil {
					fmt.Printf("ATOI Error: %v\n", err)
					continue
				}

				rowBytes,err :=  bt.Find(uint32(id))
				if err != nil {
					fmt.Print(err)
					continue
				}

				r, err := encoding.DeserializeRow(rowBytes)
				if err != nil {
					fmt.Print(err)
				}
				fmt.Printf("(%d,%s,%s)\n", r.ID, r.Username,r.Email)
			default:
				fmt.Println("Invalid Command")
		}
	}
}
