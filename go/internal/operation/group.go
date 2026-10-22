package operation

import (
	"strings"
	"unicode"

	"github.com/spf13/cobra"
)

// NewGroup creates a non-runnable command group that prints help.
func NewGroup(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
}

// FlagName converts wire names to kebab-case.
func FlagName(name string) string {
	var flag strings.Builder
	for index, character := range name {
		switch {
		case character == '_':
			flag.WriteByte('-')
		case unicode.IsUpper(character):
			if index > 0 {
				flag.WriteByte('-')
			}
			flag.WriteRune(unicode.ToLower(character))
		default:
			flag.WriteRune(character)
		}
	}
	return flag.String()
}

// Verb returns the segment before the first hyphen.
func Verb(use string) string {
	if before, _, ok := strings.Cut(use, "-"); ok {
		return before
	}
	return use
}
