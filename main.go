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
)

type ApprovedVersion struct {
	CollectionID   string `json:"collection_id"`
	ContentVersion string `json:"content_version"`
	MinCodeVersion string `json:"min_code_version"`
}

type ABL struct {
	ApprovedBooks []struct {
		CollectionID string `json:"collection_id"`
		Server       string `json:"server"`
		Style        string `json:"style"`
		TutorOnly    bool   `json:"tutor_only"`
		Books        []struct {
			UUID string `json:"uuid"`
			Slug string `json:"slug"`
		} `json:"books"`
	} `json:"approved_books"`
	ApprovedVersions []ApprovedVersion `json:"approved_versions"`
}

//will remove the first book in the ABL that matches both collectionID and contentVersion
func (a *ABL) removeBookVersion(collectionID, contentVersion string) {
	for i, v := range a.ApprovedVersions {
		if v.CollectionID == collectionID && v.ContentVersion == contentVersion {
			a.ApprovedVersions = append(a.ApprovedVersions[:i], a.ApprovedVersions[i+1:]...)
			fmt.Printf("%s %s was removed from the ABL\n", collectionID, contentVersion)
			return
		}
	}
	log.Fatalf("%s %s could not be found, nothing was removed from the ABL\n", collectionID, contentVersion)
}

//checks to see if version already in ABL, if not, adds it
func (a *ABL) addBookVersion(collectionID, contentVersion, minCodeVersion string) {
	exists := false
	for _, v := range a.ApprovedVersions {
		if v.CollectionID == collectionID && v.ContentVersion == contentVersion {
			exists = true
		}
	}
	if exists == false {
		a.ApprovedVersions = append(a.ApprovedVersions, ApprovedVersion{CollectionID: collectionID, ContentVersion: contentVersion, MinCodeVersion: minCodeVersion})
		fmt.Printf("%s %s has been added to the ABL", collectionID, contentVersion)
	} else {
		log.Fatalf("%s %s already exists in the ABL", collectionID, contentVersion)
	}
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

//func uploadABL(abl ABL) {
//	ablJson, errJson := json.Marshal(abl)
//	if errJson != nil {
//		log.Fatal("ABL could not convert to JSON:", errJson)
//	}
//	t, fileErr := ioutil.ReadFile("token.txt")
//	if fileErr != nil {
//		log.Fatal("error reading token:", fileErr)
//	}
//	ctx := context.Background()
//	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: string(t)})
//	tc := oauth2.NewClient(ctx, ts)
//	client := github.NewClient(tc)
//}

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

func pushABL() {
	fmt.Println("this doesn't work yet")
}

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
	pushPtr := flag.Bool(
		"push",
		false,
		"push local version of approved-book-list.json to github: \"./mable -push\"",
	)
	countPtr := flag.Bool(
		"count",
		false,
		"count total book versions on ABL: \"./mable -count\"",
	)
	flag.Parse()
	args := flag.Args()
	if *removePtr {
		if len(args) != 2 {
			log.Fatal("error: \"-remove\" requires 2 arguments, collection ID and content version")
		}
		if !argsError(args) {
			abl = loadABL()
			abl.removeBookVersion(args[0], args[1])
			writeABL(abl)
		}
	} else if *addPtr {
		if len(args) != 3 {
			log.Fatal("error: \"-add\" requires 3 arguments, collection ID, content version, and min code version")
		}
		if !argsError(args) {
			abl = loadABL()
			abl.addBookVersion(args[0], args[1], args[2])
			writeABL(abl)
		}
	} else if *countPtr {
		abl = loadABL()
		fmt.Printf("ABL contains %d book versions", len(abl.ApprovedVersions))
	} else if *updatePtr {
		abl = fetchABL()
		fmt.Println("most recent version of approved-book-list.json has been downloaded")
		writeABL(abl)
	} else if *pushPtr {
		abl = loadABL()
		pushABL()
	} else {
		fmt.Println("MABLE: Making the ABL Easier\nrun \"./mable -h\" for help")
	}
}
