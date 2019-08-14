package main

import (
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/quycao/gotablevalidator/tableparser"
)

func main() {
	fmt.Println("Start!")
	var errs = []error{}
	var filePath = "table.html"
	// var filePath string
	// fmt.Print("Input the html table file path: ")
	// fmt.Scanln(&filePath)

	htmlstring, err := ioutil.ReadFile(filePath)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(htmlstring)))
	doc.Find("table").EachWithBreak(func(index int, element *goquery.Selection) bool {
		err = tableparser.Init(element)
		if err != nil {
			errs = append(errs, err)
		}
		return true
	})

	for _, err := range errs {
		fmt.Println(err)
	}

	fmt.Printf("\nPress Enter Key to exit...")
	fmt.Scanln()
}
