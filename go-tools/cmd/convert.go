/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"path"
	"regexp"

	"github.com/ThierryZhou/thierryzhou.github.io/go-tools/pkg/convert"
	"github.com/spf13/cobra"
)

// convertCmd represents the convert command
var convertCmd = &cobra.Command{
	Use:   "convert",
	Short: "A tools Writen in Golang to Convert Image types",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("convert called")

		WalkAndConvert(args["src"], args["dst"])
	},
}

func init() {
	rootCmd.AddCommand(convertCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// convertCmd.PersistentFlags().String("path", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	convertCmd.Flags().String("src", "s", "source images' path")
	convertCmd.Flags().String("dst", "d", "dst images' path")
}

func WalkAndConvert(src, dst string) {

	fileInfoList, err := ioutil.ReadDir(src)
	if err != nil {
		fmt.Println(err)
	}

	for _, fileInfo := range fileInfoList {
		var fileName = fileInfo.Name()
		fmt.Println(fileName)
		isMatch, _ := regexp.MatchString(".webp$", fileName)
		if !isMatch {
			continue
		}
		webpFile := path.Join(src, fileName)
		pngFile := path.Join(src, fileName)

		convert.ConvertWebp2Png(webpFile, pngFile)
	}
}
