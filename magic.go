package main

import (
	"flag"
	"fmt"
	"log"
	miniflux "miniflux.app/client"
	"os/exec"
	"strings"
)

// CHANGE THESE PARAMETERS
var client = miniflux.New("https://rss.iscute.ovh", "username", "password")

var trivial = flag.Bool("trivial", false, "")

func Stream() <-chan *miniflux.Entry {
	c := make(chan *miniflux.Entry, 5)
	go func(pipe chan<- *miniflux.Entry) {
		defer close(pipe)
		entries := &miniflux.EntryResultSet{
			Total: 1,
			Entries: []*miniflux.Entry{
				{},
			},
		}
		offset := 0
		for entries.Entries != nil && len(entries.Entries) > 0 {
			var err error
			log.Print("fetching ", offset)
			entries, err = client.Entries(&miniflux.Filter{
				Status: miniflux.EntryStatusUnread,
				Offset: offset,
			})
			if err != nil {
				return
			}
			for _, entry := range entries.Entries {
				offset += 1
				pipe <- entry
			}
		}
	}(c)
	return c
}

func UnreadPredicate(entry miniflux.Entry) bool {
	return entry.Status == miniflux.EntryStatusUnread
}

func TrivialPredicate(entry miniflux.Entry) bool {
	return (strings.Contains(entry.Feed.FeedURL, "github.com") && (strings.Contains(entry.Title, " followed ") || strings.Contains(entry.Title, " pushed ") || strings.Contains(entry.Title, " forked "))) || (strings.Contains(entry.Feed.FeedURL, "animenewsnetwork") && strings.HasPrefix(entry.URL, "http://www.animenewsnetwork.com/review/"))
}

var Predicate func(miniflux.Entry) bool

type WebBrowser string

const (
	Dillo   WebBrowser = "dillo"
	Firefox            = "firefox"
	Elinks             = "elinks"
	Surf               = "surf"
)

const DefaultBrowser WebBrowser = Dillo

func Browser(b WebBrowser, url string) {
	cmd := exec.Command(string(b), url)
	go cmd.Run()
}

func main() {
	flag.Parse()
	if *trivial {
		Predicate = TrivialPredicate
	} else {
		Predicate = UnreadPredicate
	}
	pipe := Stream()
	for {
		entry, more := <-pipe
		if !more {
			return
		}
		if !Predicate(*entry) {
			continue
		}
		finished := false
		for !finished {
			fmt.Printf("[%v] [%v] %#v [%v] [%v]: ", entry.ID, entry.Feed.Title, entry.Title, entry.Author, entry.URL)
			var action string
			if n, err := fmt.Scan(&action); n != 1 {
				panic(err)
			}
			switch action {
			case "r", "read":
				err := client.UpdateEntries([]int64{entry.ID}, miniflux.EntryStatusRead)
				if err != nil {
					panic(err)
				}
				finished = true
			case "o", "open":
				Browser(DefaultBrowser, entry.URL)
			case "dillo":
				Browser(Dillo, entry.URL)
			case "ff", "firefox":
				Browser(Firefox, entry.URL)
			case "s", "skip":
				finished = true
			default:
				fmt.Println("hmm?")
			}
		}
	}
}
