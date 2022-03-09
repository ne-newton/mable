package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

type Book struct {
	Style string `json:"style"`
	UUID  string `json:"uuid"`
	Slug  string `json:"slug"`
}

type Version struct {
	MinCodeVersion string `json:"min_code_version"`
	Edition        int    `json:"edition"`
	CommitSha      string `json:"commit_sha"`
	CommitMetadata struct {
		CommittedAt time.Time `json:"committed_at"`
		Books       []Book `json:"books"`
	} `json:"commit_metadata"`
}

type ABL struct {
	APIVersion    int `json:"api_version"`
	ApprovedBooks []struct {
		RepositoryName string    `json:"repository_name"`
		Platforms      []string  `json:"platforms"`
		Versions       []Version `json:"versions"`
	} `json:"approved_books"`
	ApprovedVersions []interface{} `json:"approved_versions"`
}

//will remove the first book in the ABL that matches both repo, commit sha, and book slug
func (a *ABL) removeBookVersion(repositoryName, commitSha, slug string) {
	for i, ab := range a.ApprovedBooks {
		if ab.RepositoryName == repositoryName {
			for j, v := range a.ApprovedBooks[i].Versions {
				if v.CommitSha == commitSha {
					for k, b := range a.ApprovedBooks[i].Versions[j].CommitMetadata.Books {
						if b.Slug == slug {
							Books := a.ApprovedBooks[i].Versions[j].CommitMetadata.Books
							if len(Books) > 0 {
								a.ApprovedBooks[i].Versions[j].CommitMetadata.Books = append(Books[:k], Books[:k+1]...)
							} else {
								Versions := a.ApprovedBooks[i].Versions
								a.ApprovedBooks[i].Versions = append(Versions[:j], Versions[:j+1]...)
							}
						}
					}
				}
			}
			a.ApprovedVersions = append(a.ApprovedVersions[:i], a.ApprovedVersions[i+1:]...)
			fmt.Printf("%s %s of %s was removed from the ABL\n", slug, commitSha, repositoryName)
			return
		}
	}
	log.Fatalf("%s %s of %s could not be found, nothing was removed from the ABL\n", slug, commitSha, repositoryName)
}

//still needs to have version info added
//checks to see if books and version already exists, if the version exists, but not the book, it addes the book to the version
//if the version doesn't exist, it adds both. If both do exist, it creates an error
func (a *ABL) addBookVersion(repositoryName, commitSha, style, uuid, slug string) {
	exists := false
	abCount := 0
	vCount := 0
	for i, ab := range a.ApprovedBooks {
		if ab.RepositoryName == repositoryName {
			for j, v := range a.ApprovedBooks[i].Versions {
				if v.CommitSha == commitSha {
					for _, b := range a.ApprovedBooks[i].Versions[j].CommitMetadata.Books {
						if b.Slug == slug {
							exists = true
						} else {
							abCount = i
							vCount = j
						}
					}
				}
			}
		}
	}
	if exists == false {
		a.ApprovedBooks[abCount].Versions[vCount].CommitMetadata.Books = append(a.ApprovedBooks[abCount].Versions[vCount].CommitMetadata.Books, Book{Style: style, UUID: uuid, Slug: slug})
		fmt.Printf("%s %s of %s has been added to the ABL", slug, commitSha, repositoryName)
	} else {
		log.Fatalf("%s %s of %s already exists in the ABL", slug, commitSha, repositoryName)
	}
}

//adds an entirely new books to the ABL, but not a version. A version needs to be added after this step.
func (a *ABL) addNewBook() {

}

//fetches most recent ABL, unmarshalls json to struct
func fetchABL() ABL {
	ablUrl := "https://raw.githubusercontent.com/openstax/content-manager-approved-books/master/approved-book-list.json"
	req, reqErr := http.NewRequest(
		http.MethodGet,
		ablUrl,
		nil,
	)
	if reqErr != nil {
		log.Fatal("ABL request error:", reqErr)
	}
	client := &http.Client{}
	resp, respErr := client.Do(req)
	if respErr != nil {
		log.Fatal("ABL response error:", respErr)
	}
	body, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		log.Fatal("ABL read error:", readErr)
	}
	var abl ABL
	if err := json.Unmarshal(body, &abl); err != nil {
		log.Fatal("Remote ABL could not be unmarshalled to json:", err)
	}
	return abl
}

//needs to be updated for newest version of ABL

//checks if cl arguments are valid
func argsError(args []string) bool {
	clError := false
	validColID := regexp.MustCompile(`col\d{5}`)
	validVersion := regexp.MustCompile(`\d+\.\d+\.?\d*`)
	validMinCode := regexp.MustCompile(`\d{8}\.\d{6}`)
	if !validColID.MatchString(args[0]) {
		clError = true
		log.Fatalf("error: %s is not a valid collection ID", args[0])
	}
	if !validVersion.MatchString(args[1]) {
		clError = true
		log.Fatalf("error: %s is not a valid content version", args[1])
	}
	if len(args) < 3 {
		return clError
	}
	if !validMinCode.MatchString(args[2]) {
		clError = true
		log.Fatalf("error: %s is not a valid minimum code version", args[2])
	}
	return clError
}

//if local ABL present, load file, otherwise fetch ABL from github
func loadABL() ABL {
	var abl ABL
	if _, err := os.Stat("approved-book-list.json"); err == nil {
		o, openErr := os.Open("approved-book-list.json")
		if openErr != nil {
			log.Fatal("Couldn't open file:", openErr)
		}
		defer o.Close()
		f, readErr := ioutil.ReadAll(o)
		if readErr != nil {
			log.Fatal("Couldn't read file:", readErr)
		}
		if jsonErr := json.Unmarshal(f, &abl); jsonErr != nil {
			log.Fatal("Local ABL could not be unmarshalled to json:", err)
		}
	} else if os.IsNotExist(err) {
		abl = fetchABL()
	} else {
		log.Print(err)
	}
	return abl
}

func writeABL(abl ABL) {
	ablJson, jsonErr := json.MarshalIndent(abl, "", "  ")
	if jsonErr != nil {
		log.Fatal("error creating json:", jsonErr)
	}
	if err := ioutil.WriteFile("approved-book-list.json", ablJson, 0644); err != nil {
		log.Fatal("error writing json:", err)
	}
}

//flags need to be updated
func main() {
	var abl ABL
	removePtr := flag.Bool(
		"remove",
		false,
		"remove a book version from the ABL: \"./mable -remove {collection ID} {content version}\"",
	)
	addPtr := flag.Bool(
		"add",
		false,
		"add a book to the ABL: \"./mable -add {collection ID} {content version} {min code version}\"",
	)
	updatePtr := flag.Bool(
		"update",
		false,
		"download the most recent version of the approved-book-list.json: \"./mable -update\"",
	)
	countPtr := flag.Bool(
		"count",
		false,
		"count total book versions on ABL: \"./mable -count\"",
	)
	flag.Parse()
	args := flag.Args()
	if *removePtr {
		if len(args) != 3 {
			log.Fatal("error: \"-remove\" requires 3 arguments, repository name, commit SHA, and slug")
		}
		if !argsError(args) {
			abl = loadABL()
			abl.removeBookVersion(args[0], args[1], args[2])
			writeABL(abl)
		}
	} else if *addPtr {
		if len(args) != 5 {
			log.Fatal("error: \"-add\" requires 5 arguments, repository name, commit SHA, style, Book UUID, and slug")
		}
		if !argsError(args) {
			abl = loadABL()
			abl.addBookVersion(args[0], args[1], args[2], args[3], args[4])
			writeABL(abl)
		}
	} else if *countPtr {
		abl = loadABL()
		fmt.Printf("ABL contains %d book versions", len(abl.ApprovedVersions))
	} else if *updatePtr {
		abl = fetchABL()
		fmt.Println("most recent version of approved-book-list.json has been downloaded")
		writeABL(abl)
	} else {
		fmt.Println("MABLE: Making the ABL Easier\nrun \"./mable -h\" for help")
	}
}
