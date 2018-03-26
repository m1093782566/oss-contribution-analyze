package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"

	"github.com/golang/glog"
)

// GithubPersonProfile is basic github person information.
type GithubPersonProfile struct {
	ID       string
	WorksFor string
	Email    string
}

// GithubContributor carries the contributions of each person
type GithubContributor struct {
	Num    int
	Person *GithubPersonProfile
}

// We only care about the following companies - they are all in lower case.
var Companies = []string{"google", "red hat", "microsoft", "huawei"}

func (p *GithubPersonProfile) ScrapeProfileById(id string) error {
	url := "https://github.com/" + id
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	//First login github.com
	//Press F12 to get the detailed cookie info
	//Fill in the cookie
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:58.0) Gecko/20100101 Firefox/58.0")
	req.Header.Add("Referer", url)
	req.Header.Add("Cookie", "logged_in=yes; _octo=GH1.1.709081642.1513934084; _ga=GA1.2.1671139169.1513934084; _gh_sess=dVVI***; tz=Australia%2FPerth; user_session=_phQVf***; __Host-user_session_same_site=_phQVfL**; dotcom_user=***; _gat=1")

	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	doc, err := goquery.NewDocumentFromResponse(res)
	if err != nil {
		return err
	}
	wf := doc.Find("li[itemprop='worksFor']").Text()
	em := doc.Find("li[itemprop='email']").Text()
	p.WorksFor = wf
	p.Email = em
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
				Num:    num,
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
