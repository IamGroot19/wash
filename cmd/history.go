package cmd

import (
	"io"
	"os"
	"strconv"

	"github.com/Benchkram/errz"
	cmdutil "github.com/puppetlabs/wash/cmd/util"
	"github.com/spf13/cobra"
)

func historyCommand() *cobra.Command {
	historyCmd := &cobra.Command{
		Use:     "history [-f] [<id>]",
		Aliases: []string{"whistory"},
		Short:   "Prints the wash command history, or journal of a particular item",
		Long: `Wash maintains a history of commands executed through it. Print that command history, or specify an
<id> to print a log of activity related to a particular command.`,
		Args: cobra.MaximumNArgs(1),
		RunE: toRunE(historyMain),
	}
	historyCmd.Flags().BoolP("follow", "f", false, "Follow new updates")
	return historyCmd
}

func printJournalEntry(index string, follow bool) error {
	idx, err := strconv.Atoi(index)
	if err != nil {
		return err
	}

	conn := cmdutil.NewClient()
	// Translate from 1-indexing for history entries
	rdr, err := conn.ActivityJournal(idx-1, follow)
	if err != nil {
		return err
	}
	defer func() {
		errz.Log(rdr.Close())
	}()

	_, err = io.Copy(os.Stdout, rdr)
	return err
}

func printHistory(follow bool) error {
	conn := cmdutil.NewClient()
	history, err := conn.History(follow)
	if err != nil {
		return err
	}

	// Use 1-indexing for history entries
	indexColumnLength := len(strconv.Itoa(len(history)))
	formatStr := "%" + strconv.Itoa(indexColumnLength) + "d  %s  %s\n"
	i := 0
	for item := range history {
		cmdutil.Printf(formatStr, i+1, item.Start.Format("2006-01-02 15:04"), item.Description)
		i++
	}
	return nil
}

func historyMain(cmd *cobra.Command, args []string) exitCode {
	follow, err := cmd.Flags().GetBool("follow")
	if err != nil {
		panic(err.Error())
	}

	if len(args) > 0 {
		err = printJournalEntry(args[0], follow)
	} else {
		err = printHistory(follow)
	}

	if err != nil {
		cmdutil.ErrPrintf("%v\n", err)
		return exitCode{1}
	}

	return exitCode{0}
}
