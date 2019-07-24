package cmd

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/danielkvist/beagle/logger"
	"github.com/danielkvist/beagle/sites"

	"github.com/spf13/cobra"
)

var (
	agent      string
	csvFile    string
	user       string
	goroutines int
)

func init() {
	rootCmd.PersistentFlags().StringVarP(&agent, "agent", "a", "Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:67.0) Gecko/20100101 Firefox/67.0", "user agent")
	rootCmd.PersistentFlags().StringVar(&csvFile, "csv", "./urls.csv", ".csv file with the URLs to parse and check")
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "me", "username you want to search for")
	rootCmd.PersistentFlags().IntVarP(&goroutines, "goroutines", "g", 1, "number of goroutines")
}

var rootCmd = &cobra.Command{
	Use:   "beagle",
	Short: "",
	RunE: func(cmd *cobra.Command, args []string) error {
		siteList, err := sites.Parse(csvFile)
		if err != nil {
			return err
		}

		l := logger.New(os.Stdout, goroutines)
		c := &http.Client{}

		sema := make(chan struct{}, goroutines)
		var wg sync.WaitGroup

		for _, s := range siteList {
			wg.Add(1)
			sema <- struct{}{}

			go func(site *sites.Site) {
				defer func() {
					<-sema
					wg.Done()
				}()

				site.ReplaceURL("$", user)
				_, statusCode, _ := check(c, site.URL, agent) // FIXME:
				l.Println(formatMsg(site.Name, site.URL, statusCode))
			}(s)
		}

		wg.Wait()
		l.Stop()
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}

func check(c *http.Client, url string, agent string) (string, int, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("User-Agent", agent)

	resp, err := c.Do(req)
	if err != nil {
		return "", 0, err
	}

	return resp.Status, resp.StatusCode, nil
}

func formatMsg(name string, url string, status int) string {
	if status != http.StatusOK {
		return fmt.Sprintf("NO %s %s", name, url)
	}
	return fmt.Sprintf("OK %s %s", name, url)
}
