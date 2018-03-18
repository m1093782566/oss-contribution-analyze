package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/golang/glog"
)

// GithubPersonProfile is basic github person information.
type GithubPersonProfile struct {
	ID string
	WorksFor string
	Email string
}

// GithubContributor carries the contributions of each person
type GithubContributor struct {
	Num int
	Person *GithubPersonProfile
}

// We only care about the following companies - they are all in lower case.
var Companies = []string{"google", "red hat", "microsoft", "huawei"}

func (p *GithubPersonProfile)ScrapeProfileById(id string) error {
	url := "https://github.com/" + id
	doc, err := goquery.NewDocument(url)
	if err != nil {
		return err
	}
	wf := doc.Find("li[itemprop='worksFor']").Text()
	// TODO: should login first to fetch email
	// em := doc.Find("li[itemprop='email']").Text()
	p.WorksFor = wf
	return nil
}

func main() {
	// CHANGELOG-X.Y.md should be in the same directory as this execute file
	releaseVersions := []string{"1.7", "1.8", "1.9", "1.10"}
	for _, version := range releaseVersions {
		expStr := fmt.Sprintf(`grep -Po "(#\d+).*(@[a-zA-Z0-9/-]+)" CHANGELOG-%s.md | sed  's/].*\[/,/' | sed 's/ //' | sort | uniq | sed 's/#[0-9]*,//' | grep -Po @[a-zA-Z0-9/-]+ | sort | uniq -c`, version)
		cmd := exec.Command("sh", "-c", expStr)
		stdoutStderr, err := cmd.CombinedOutput()
		if err != nil {
			glog.Fatal(err)
		}

		// output should be in the format of
		// 7 @abc
		// 2 @def
		// ...
		mulLines := bytes.Split(stdoutStderr, []byte{'\n'})
		companyContributions := make(map[string]int)
		for _, line := range mulLines {
			// Ignore empty lines
			if len(line) == 0 {
				continue
			}
			strs := strings.Split(string(line), " @")
			if len(strs) != 2 {
				glog.Errorf("Error format of %s, should be in the format of <integer, @abc>", line)
				continue
			}
			val := strings.Trim(strs[0], " ")
			num, err := strconv.Atoi(val)
			if err != nil {
				glog.Errorf("Error contributions number [%s]", val)
				continue
			}
			contributor := &GithubContributor{
				Num: num,
				Person: &GithubPersonProfile{ID: strs[1]},
			}
			// TODO: work in concurrent
			if err := contributor.Person.ScrapeProfileById(contributor.Person.ID); err != nil {
				glog.Errorf("Error scrape profile by ID, %v", err)
				continue
			}
			// Filter {Google, Red Hat, Microsoft, Huawei}
			for _, company := range Companies {
				worksFor := strings.ToLower(contributor.Person.WorksFor)
				if index := strings.Index(worksFor, company); index != -1 {
					companyContributions[company] += contributor.Num
				}
			}
		}
		// Output result
		fmt.Printf("Release: %s\n", version)
		for company, contributions := range companyContributions {
			fmt.Printf("%s: %d\n", company, contributions)
		}
	}
}
