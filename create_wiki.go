package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"

	"golang.org/x/net/html"
)

// Beginner's foray in Go.
// "pointers to interfaces are almost never useful."

// Types

// TableAwareWriter is a type representing a writer that can write to a table
type TableAwareWriter struct {
	MaxCols    int
	CurrentCol int
	Output     io.Writer
}

// Methods

// Write is a method that writes a byte array in a table-aware way.
func (taw *TableAwareWriter) Write(p []byte) (int, error) {
	writeBytes := []byte("")
	writeBytes = append(writeBytes, " | "...)
	writeBytes = append(writeBytes, p...)
	//if last column, end with pipe and newline
	if taw.CurrentCol >= (taw.MaxCols - 1) {
		taw.CurrentCol = 0
		writeBytes = append(writeBytes, " |\n"...)
	} else {
		taw.CurrentCol++
	}
	return taw.Output.Write(writeBytes)
}

// WriteString is a method that writes a string in a table-aware way.
//func (taw TableAwareWriter) WriteString(p string) (int, error) {
//	return taw.Output.Write([]byte(p))
//}

// WriteTableHeader is a method that writes a byte array in a table-aware way.
func (taw TableAwareWriter) WriteTableHeader() (int, error) {

	firstRow := ""
	secondRow := ""
	for i := 0; i < taw.MaxCols; i++ {
		firstRow += "| "
		secondRow += "| --- "
	}
	firstRow += "|\n"
	secondRow += "|\n"

	out := firstRow + secondRow
	return taw.Output.Write([]byte(out))

}

// WriteHeader is a method that writes a byte array in a table-aware way.
func (taw TableAwareWriter) WriteHeader(s string) (int, error) {

	return taw.Output.Write([]byte(s))

}

// Provider is a type representing a Terraform provider
type Provider struct {
	Name   string
	Output io.Writer
}

// Methods

// SetupParse is a method to get ready for parsing HTML.
func (p *Provider) SetupParse() (*html.Node, error) {
	providerURL := "https://www.terraform.io/docs/providers/" + strings.ToLower(p.Name) + "/"

	resp, _ := http.Get(providerURL)
	bytes, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	htmlNode, err := html.Parse(strings.NewReader(string(bytes)))
	if err != nil {
		return nil, err
	}

	return htmlNode, nil
}

// WalkNode is a method to walk an HTML node looking for documentation link ul.
func (p *Provider) WalkNode(n *html.Node) bool {

	keepGoing := true

	if n.Type == html.ElementNode && n.Data == "ul" {
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == "nav docs-sidenav" {
				processProviderIndex(n, p.Output, 0, strings.ToLower(p.Name))
				keepGoing = false
				break
			}
		}

		if !keepGoing {
			return false
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if !p.WalkNode(c) {
			return false
		}
	}

	return true
}

// Process is a method to process a provider.
func (p *Provider) Process() error {
	// create a provider page
	err := os.Mkdir(strings.ToLower(p.Name), os.FileMode(int(0755)))
	if err != nil {
		return err
	}

	err = os.Chdir(strings.ToLower(p.Name))
	if err != nil {
		return err
	}

	f, err := os.Create(strings.ToLower(p.Name) + ".md")
	if err != nil {
		return err
	}
	defer f.Close()
	mainPage := &TableAwareWriter{2, 0, f}
	p.Output = mainPage
	mainPage.WriteHeader("# " + p.Name + "\n\n")
	mainPage.WriteHeader("[Home](Home)\n\n")
	mainPage.WriteTableHeader()

	htmlNode, err := p.SetupParse()
	if err != nil {
		return err
	}

	p.WalkNode(htmlNode)

	return nil
}

// Functions

func writeMDLink(n *html.Node, f io.Writer, replaceURL string) {
	if n.Type == html.ElementNode && n.Data == "a" {
		linkText := ""
		href := ""
		if n.FirstChild != nil {
			linkText = n.FirstChild.Data
		}
		for _, a := range n.Attr {
			if a.Key == "href" {
				href = a.Val
				if replaceURL != "" {
					href = replaceURL
				}
				if href == "#" {
					f.(*TableAwareWriter).WriteHeader("# " + linkText + "\n\n")
					f.(*TableAwareWriter).WriteTableHeader()
				} else if linkText != "" {
					if strings.HasPrefix(href, "/docs") {
						href = "https://www.terraform.io" + href
					}
					io.WriteString(f, "["+linkText+"]("+href+")")
				}
				break
			}
		}
	}
}

func processProviderIndex(n *html.Node, f io.Writer, depth int, name string) {

	if n.Type == html.ElementNode && n.Data == "a" {
		writeMDLink(n, f, "")
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if depth == 0 && c.FirstChild != nil && c.FirstChild.NextSibling != nil && c.FirstChild.NextSibling.Data == "a" {
			if !blacklist(c.FirstChild.NextSibling.FirstChild.Data) {
				filename := name + "_" + strings.ToLower(strings.Replace(c.FirstChild.NextSibling.FirstChild.Data, " ", "_", -1))
				writeMDLink(c.FirstChild.NextSibling, f, filename)
				var err error
				writeFile, err := os.Create(filename + ".md") //need to md for filename but not for link to file
				defer writeFile.Close()
				check(err)
				writeFile.WriteString("[" + name + "](" + name + ")\n")
				tableWriter := &TableAwareWriter{2, 0, writeFile}
				processProviderIndex(c, tableWriter, depth+1, name)
			}
		} else {
			processProviderIndex(c, f, depth+1, name)
		}
	}
}

func blacklist(check string) bool {
	switch check {
	case "All Providers":
		return true
	case "AWS Provider":
		return true
	}
	return false
}

func check(e error) {
	if e != nil {
		log.Fatal(e)
		panic(e)
	}
}

func createWiki() {
	err := os.Mkdir("wiki", os.FileMode(int(0755)))
	check(err)
	os.Chdir("wiki")

	p := Provider{"AWS", nil}
	check(p.Process())

	os.Chdir("..")

	p = Provider{"AzureRM", nil}
	check(p.Process())

	os.Chdir("..")

	p = Provider{"Google", nil}
	check(p.Process())

	os.Chdir("..")

	p = Provider{"Alicloud", nil}
	check(p.Process())

	os.Chdir("..")

	p = Provider{"OpenStack", nil}
	check(p.Process())
	//os.Getwd()
}

func main() {
	createWiki()
}
