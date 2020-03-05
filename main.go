package main

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/alexsasharegan/dotenv"
	"github.com/kelseyhightower/envconfig"
	"github.com/mattn/go-mastodon"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type (
	Config struct {
		Mastodon struct {
			Server      string `envconfig:"MASTODON_SERVER" required:"true"`
			AccessToken string `envconfig:"MASTODON_ACCESSTOKEN" required:"true"`
		}
		WaitSec int `envconfig:"WAIT_SEC" required:"true"`
		Shindan struct {
			IDs  string `envconfig:"SHINDAN_IDS" required:"true"`
			Name string `envconfig:"SHINDAN_NAME" required:"true"`
		}
	}
)

func main() {
	config := loadEnv()
	client := mastodon.NewClient(&mastodon.Config{
		Server:      "https://" + config.Mastodon.Server,
		AccessToken: config.Mastodon.AccessToken,
	})

	ids := strings.Split(config.Shindan.IDs, ",")
	for i, id := range ids {
		result := fetch(strings.TrimSpace(id), config.Shindan.Name)
		if _, err := client.PostStatus(context.TODO(), &mastodon.Toot{Status: result}); err != nil {
			log.Printf("mastodon post error: %v", err)
		}

		if i < len(ids)-1 {
			time.Sleep(time.Duration(config.WaitSec) * time.Second)
		}
	}
}

func loadEnv() Config {
	if err := dotenv.Load(); err != nil {
		log.Fatalf("error .env: %v", err)
	}
	var c Config
	envconfig.MustProcess("", &c)
	return c
}

func fetch(id string, name string) string {
	uri := "https://shindanmaker.com/" + id

	form := url.Values{}
	form.Add("u", name)
	resp, err := http.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
	if err != nil {
		log.Println("fetch: %v, error: %v", id, err)
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return ""
	}

	var text = ""
	var selection *goquery.Selection
	selection = doc.Find("#copy_text")
	if len(selection.Nodes) != 0 {
		text = selection.Text()
	} else {
		selection = doc.Find("#copy_text_140")
		if len(selection.Nodes) != 0 {
			text = selection.Text()
		}
	}

	log.Printf("fetch: %v, status: %v, result: %v", id, resp.StatusCode, text)
	return text
}
