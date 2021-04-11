package main

import (
	"context"
	"github.com/PuerkitoBio/goquery"
	"github.com/alexsasharegan/dotenv"
	"github.com/antchfx/htmlquery"
	"github.com/kelseyhightower/envconfig"
	"github.com/mattn/go-mastodon"
	"log"
	"net/http"
	"net/http/cookiejar"
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

func httpClient() (*http.Client, error) {
	cookie, err := cookiejar.New(nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Jar: cookie,
	}
	return client, nil
}

func fetch(id string, name string) string {
	uri := "https://shindanmaker.com/" + id

	client, err := httpClient()
	if err != nil {
		log.Println("fetch: %v, error: %v", id, err)
		return ""
	}
	res, err := client.Get(uri)
	if err != nil {
		log.Println("fetch: %v, error: %v", id, err)
		return ""
	}
	defer res.Body.Close()
	parsed, err := htmlquery.Parse(res.Body)
	if err != nil {
		log.Println("fetch: %v, error: %v", id, err)
		return ""
	}
	tokenInput := htmlquery.FindOne(parsed, "//input[@name='_token']/@value")
	hiddenNameInput := htmlquery.FindOne(parsed, "//input[@name='hiddenName']/@value")
	if tokenInput == nil || hiddenNameInput == nil {
		log.Println("fetch: %v, error: %v", id, "tokenInput == nil || hiddenNameInput == nil")
		return ""
	}
	token := tokenInput.FirstChild.Data
	hiddenName := hiddenNameInput.FirstChild.Data

	form := url.Values{}
	form.Add("_token", token)
	form.Add("name", name)
	form.Add("hiddenName", hiddenName)
	resp, err := client.Post(uri, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
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
	selection = doc.Find("#copy-textarea")
	if len(selection.Nodes) != 0 {
		text = selection.Text()
	} else {
		selection = doc.Find("#copy-textarea-140")
		if len(selection.Nodes) != 0 {
			text = selection.Text()
		}
	}

	log.Printf("fetch: %v, status: %v, result: %v", id, resp.StatusCode, text)
	return text
}
