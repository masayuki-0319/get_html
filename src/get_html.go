package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"get_html/config"

	"github.com/PuerkitoBio/goquery"
)

var imageHostPattern = config.Config.ImgURLPattern
var resultDir = config.Config.StoreDir

func main() {
	// _. CLI からの引数を検証
	flag.Parse()
	requestURL := validateArgs(flag.Args())

	// _. DOM を取得
	doc := getDoc(requestURL)

	// 1. 保存用のディレクトリ用意
	dirName := makeDirectory(doc)

	// 2. 画像 URL の書き出しファイルを生成
	outputFilePath := dirName + "/output.txt"
	file := touchOutputFile(outputFilePath)
	defer file.Close()
	defer os.Remove(outputFilePath)

	// 3. 画像 URL 一覧を保存
	writeURLs(doc, file)

	// _. 読み込みのため，ファイルの先頭に戻す
	file.Seek(0, 0)

	// 4. txt 中の URL から画像ダウンロード
	downloadURLs(file, dirName)
}

func downloadURLs(file *os.File, dirName string) {
	scanner := bufio.NewScanner(file)

	// １行毎に読み込み
	i := 1
	for scanner.Scan() {
		url := scanner.Text()
		filename := dirName + "/" + strconv.Itoa(i)

		fmt.Println(scanner.Text())
		err := downloadFile(url, filename + ".jpg")
		if err != nil {
			rep := regexp.MustCompile(`\.(jpg)$`)
			url := rep.ReplaceAllString(url, ".png")

			fmt.Println(err)
			fmt.Println("Re request by changing extension")
			fmt.Println(url)
			fmt.Println("------------------------------")

			downloadFile(url, filename + ".png")
		}
		i++
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
}

func downloadFile(URL, fileName string) error {
	client := http.Client{
		Timeout: 1 * time.Second,
	}
	response, err := client.Get(URL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		return errors.New("Received non 200 response code")
	}

	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	return nil
}

func writeURLs(doc *goquery.Document, file *os.File) {
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		url, bool := s.Attr("data-src")
		if bool != true {
			fmt.Println("URL が見つかりません")
		}
		// url, _ := s.Attr("src")
		if isMatchHost(url) != true {
			return
		}

		_, _ = file.WriteString(url + "\n")
	})
}

func isMatchHost(str string) (bool) {
    result, _ := regexp.MatchString(imageHostPattern, str)
	return result
}

func touchOutputFile(filePath string) *os.File {
	// 後々読み込むため RDWR
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0664)
	if err != nil {
		panic(err)
	}

	return file
}

func makeDirectory(doc *goquery.Document) (string) {
	actualHeader := doc.Find("h1").Text()
	sanitizeRep := regexp.MustCompile(`^\(.+\)\s`)
	header := sanitizeRep.ReplaceAllString(actualHeader, "")
	fmt.Printf("Title: %s\n", header)

	exe, _ := os.Getwd()
	absStorePath := exe + "/" + resultDir + "/" + header

	err := os.Mkdir(absStorePath, 0777)
	returnDirName := absStorePath
	if err != nil {
		fmt.Println(err)

		dirName := filepath.Dir(absStorePath)
		err := os.Mkdir(dirName, 0777)
		returnDirName = dirName
		if err != nil {
			panic(err)
		}
	}

	fmt.Println(filepath.Abs(returnDirName))
	return returnDirName
}

func getDoc(url *url.URL) (*goquery.Document) {
	res, err := http.Get(url.String())
	if err != nil {
		fmt.Println(err)
		panic("Fail get HTML")
	}
	defer res.Body.Close()

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		panic(err)
	}

	return doc
}

func validateArgs(args []string) (*url.URL) {
	if len(args) == 0 {
		panic("Error: Set target URL for first args.")
	}

	requestURL, err := url.ParseRequestURI(args[0])
	if err != nil {
		panic(err)
	}
	return requestURL
}
