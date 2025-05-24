package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

// Anime represents the extracted anime details.
type Anime struct {
	Title         string `json:"title"`
	OriginalTitle string `json:"originalTitle"`
	Description   string `json:"description"`
	Image         string `json:"image"`
	URL           string `json:"url"`
	Rating        string `json:"rating"`
	Episodes      string `json:"episodes"`
	Duration      string `json:"duration"`
	Type          string `json:"type"`
	Genres        string `json:"genres"`
}

// fetchAnimeDetails extracts anime details from the anime page.
func fetchAnimeDetails(animeURL string) (Anime, error) {
	req, err := http.NewRequest("GET", animeURL, nil)
	if err != nil {
		return Anime{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return Anime{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Anime{}, err
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return Anime{}, err
	}

	img := doc.Find(".poster img")
	image, _ := img.Attr("src")
	title, _ := img.Attr("alt")
	description := strings.TrimSpace(doc.Find(".desc").Text())

	getField := func(label string) string {
		var val string
		// Find each div inside .detail > div (there are two main divs)
		doc.Find(".detail > div").EachWithBreak(func(i int, container *goquery.Selection) bool {
			found := false
			container.ChildrenFiltered("div").EachWithBreak(func(j int, s *goquery.Selection) bool {
				text := s.Contents().First().Text()
				if strings.Contains(text, label+":") {
					val = strings.TrimSpace(s.Find("span").Text())
					found = true
					return false
				}
				return true
			})
			return !found
		})
		return val
	}

	rating := strings.TrimSpace(doc.Find(".info .rating").Text())
	episodes := getField("Episodes")
	duration := getField("Duration")
	genres := getField("Genres")
	animeType := strings.TrimSpace(doc.Find(".info span b").First().Text())
	originalTitle, _ := doc.Find(".title").Attr("data-jp")

	return Anime{
		Title:         title,
		OriginalTitle: originalTitle,
		Description:   description,
		Image:         image,
		URL:           animeURL,
		Rating:        rating,
		Episodes:      episodes,
		Duration:      duration,
		Type:          animeType,
		Genres:        genres,
	}, nil
}

// getCurrentlyWatching fetches the current anime user is watching.
func getCurrentlyWatching(w http.ResponseWriter, r *http.Request) {
	profileURL := "https://animekai.to/user/m3nvy/profile" // User Profile

	req, err := http.NewRequest("GET", profileURL, nil)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("User-Agent", "Mozilla/5.0")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Failed to fetch profile", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Profile request failed", http.StatusInternalServerError)
		return
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		http.Error(w, "Failed to parse profile HTML", http.StatusInternalServerError)
		return
	}

	var anime Anime
	foundAnime := false

	doc.Find("div.status").EachWithBreak(func(i int, s *goquery.Selection) bool {
		link := s.Find("a")
		href, exists := link.Attr("href")
		if exists {
			animeURL := "https://animekai.to" + href

			animeData, err := fetchAnimeDetails(animeURL)
			if err != nil {
				return true // continue searching if error
			}

			anime = animeData
			foundAnime = true
			return false // stop iteration
		}
		return true
	})

	if !foundAnime {
		http.Error(w, "No currently watching anime found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anime)
}

// corsMiddleware adds CORS headers for frontend access.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", getCurrentlyWatching)

	handler := corsMiddleware(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback for local dev
	}

	log.Println("Server running on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", handler))
}
