/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"strings"
	v1 "sxsk/pkg/v1"

	"github.com/spf13/cobra"
)

// v1Cmd represents the v1 command
var v1Cmd = &cobra.Command{
	Use:     "v1",
	Args:    cobra.ExactArgs(1),
	Example: `sxsk v1 [岗位代码]`,
	Short:   "v1版本的报考人数查询",
	Long:    `通过 https://sn.huatu.com 提供的接口查询各岗位的报名信息`,
	Run: func(cmd *cobra.Command, args []string) {
		v1.Exec(strings.Split(args[0], ","))
	},
}

func init() {
	rootCmd.AddCommand(v1Cmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// v1Cmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// v1Cmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
