package sql

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

type SQLPair struct {
	Query  string
	Result string
}

func BuildSQLPair(sql_template, sqldir, resultdir string) (map[string]SQLPair, error) {
	sqlPath, err := filepath.Abs(sqldir)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed generate absolute file path of %s", sqldir))
	}
	resultPath, err := filepath.Abs(resultdir)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed generate absolute file path of %s", resultdir))
	}

	items, _ := ioutil.ReadDir(sqlPath)
	ans := make(map[string]SQLPair)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else if strings.HasSuffix(item.Name(), ".sql") {
			key := strings.TrimSuffix(item.Name(), ".sql")
			var pair SQLPair
			if p, found := ans[key]; !found {
				pair = SQLPair{}
			} else {
				pair = p
			}
			sql, err := ioutil.ReadFile(sqlPath + "/" + item.Name())
			if err != nil {
				return nil, err
			}
			pair.Query = fmt.Sprintf(sql_template, string(sql))
			ans[key] = pair
		}
	}
	items, _ = ioutil.ReadDir(resultPath)
	for _, item := range items {
		if item.IsDir() {
			continue
		} else if strings.HasSuffix(item.Name(), ".result") {
			key := strings.TrimSuffix(item.Name(), ".result")
			var pair SQLPair
			if p, found := ans[key]; !found {
				pair = SQLPair{}
			} else {
				pair = p
			}
			result, err := ioutil.ReadFile(resultPath + "/" + item.Name())
			if err != nil {
				return nil, err
			}
			pair.Result = string(result)
			ans[key] = pair
		}
	}
	return ans, nil
}
