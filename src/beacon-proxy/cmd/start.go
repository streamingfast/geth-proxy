package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"net/http"
)

var startWait = &cobra.Command{
	Use:   "start",
	Short: "starts listening for requests from beacon",
	Run: func(cmd *cobra.Command, args []string) {
		awaitResponse()
	},
}

func init() {
	rootCmd.AddCommand(startWait)
}

func hello(w http.ResponseWriter, req *http.Request) {
	fmt.Println("Hello")
}

func awaitResponse() {
	http.HandleFunc("/hello", hello)
	http.ListenAndServe(":8080", nil)
}
