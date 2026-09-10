package vserverclient

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

func PaginationParams(cmd *cobra.Command, pageParam, sizeParam string, defaultPageSize int) map[string]string {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	return map[string]string{
		pageParam: fmt.Sprintf("%d", page),
		sizeParam: fmt.Sprintf("%d", pageSize),
	}
}

func ValidatedPaginationParams(cmd *cobra.Command, pageParam, sizeParam string) (map[string]string, error) {
	page, _ := cmd.Flags().GetInt("page")
	pageSize, _ := cmd.Flags().GetInt("page-size")
	if page < 1 || pageSize < 1 {
		return nil, fmt.Errorf("--page and --page-size must be positive")
	}
	return map[string]string{
		pageParam: strconv.Itoa(page),
		sizeParam: strconv.Itoa(pageSize),
	}, nil
}
