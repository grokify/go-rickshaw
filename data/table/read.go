package table

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/grokify/simplego/encoding/csvutil"
	"github.com/grokify/simplego/type/stringsutil"
	"github.com/pkg/errors"
)

var debugReadCSV = false // should not need to use this.

// ReadFiles reads in a list of delimited files and returns a merged `Table` struct.
// An error is returned if the columns count differs between files.
func ReadFiles(filenames []string, comma rune, hasHeader bool) (Table, error) {
	tbl := NewTable()
	for i, filename := range filenames {
		tblx, err := ReadFile(filename, comma, hasHeader)
		if err != nil {
			return tblx, err
		}
		if i > 0 && len(tbl.Columns) != len(tblx.Columns) {
			return tbl, fmt.Errorf("csv column count mismatch earlier files count [%d] file [%s] count [%d]",
				len(tbl.Columns), filename, len(tblx.Columns))
		}
	}
	return tbl, nil
}

// ReadFile reads in a delimited file and returns a `Table` struct.
func ReadFile(filename string, comma rune, hasHeader bool) (Table, error) {
	tbl := NewTable()
	csvReader, f, err := csvutil.NewReader(filename, comma, false)
	if err != nil {
		return tbl, err
	}
	defer f.Close()
	if debugReadCSV {
		i := -1
		for {
			line, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return tbl, err
			}
			i++
			if i == 0 && hasHeader {
				tbl.Columns = line
				continue
			}
			tbl.Records = append(tbl.Records, line)
			if i > 2500 {
				fmt.Printf("[%v] %v\n", i, strings.Join(line, ","))
			}
		}
	} else {
		lines, err := csvReader.ReadAll()
		if err != nil {
			return tbl, err
		}
		byteOrderMarkAsString := string('\uFEFF')
		if len(lines) > 0 && len(lines[0]) > 0 &&
			strings.HasPrefix(lines[0][0], byteOrderMarkAsString) {
			lines[0][0] = strings.TrimPrefix(lines[0][0], byteOrderMarkAsString)
		}
		if hasHeader {
			tbl.LoadMergedRows(lines)
		} else {
			tbl.Records = lines
		}
	}
	return tbl, nil
}

/*
func ReadMergeFilterCSVFiles(inPaths []string, outPath string, inComma rune, inStripBom bool, andFilter map[string]stringsutil.MatchInfo) (DocumentsSet, error) {
	//data := JsonRecordsInfo{Records: []map[string]string{}}
	data := NewDocumentsSet()

	for _, inPath := range inPaths {
		reader, inFile, err := csvutil.NewReader(inPath, inComma, inStripBom)
		if err != nil {
			return data, err
		}

		csvHeader := csvutil.CSVHeader{}
		j := -1

		for {
			line, err := reader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return data, err
			}
			j++

			if j == 0 {
				csvHeader.Columns = line
				continue
			}
			match, err := csvHeader.RecordMatch(line, andFilter)
			if err != nil {
				return data, err
			}
			if !match {
				continue
			}

			mss := csvHeader.RecordToMSS(line)
			data.Documents = append(data.Documents, mss)
		}
		err = inFile.Close()
		if err != nil {
			return data, err
		}
	}
	data.Inflate()
	return data, nil
}
*/
/*
func MergeFilterCSVFilesToJSON(inPaths []string, outPath string, inComma rune, inStripBom bool, perm os.FileMode, andFilter map[string]stringsutil.MatchInfo) error {
	data, err := ReadMergeFilterCSVFiles(inPaths, outPath, inComma, inStripBom, andFilter)
	if err != nil {
		return err
	}
	bytes, err := jsonutil.MarshalSimple(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(outPath, bytes, perm)
}
*/

func ReadCSVFilesSingleColumnValuesString(files []string, sep rune, hasHeader bool, col uint, condenseUniqueSort bool) ([]string, error) {
	values := []string{}
	for _, file := range files {
		fileValues, err := ReadCSVFileSingleColumnValuesString(
			file, sep, hasHeader, col, false)
		if err != nil {
			return values, err
		}
		values = append(values, fileValues...)
	}
	if condenseUniqueSort {
		values = stringsutil.SliceCondenseSpace(values, true, true)
	}
	return values, nil
}

func ReadCSVFileSingleColumnValuesString(filename string, sep rune, hasHeader bool, col uint, condenseUniqueSort bool) ([]string, error) {
	tbl, err := ReadFile(filename, sep, hasHeader)
	if err != nil {
		return []string{}, err
	}
	values := []string{}
	for _, row := range tbl.Records {
		if len(row) > int(col) {
			values = append(values, row[col])
		}
	}
	if condenseUniqueSort {
		values = stringsutil.SliceCondenseSpace(values, true, true)
	}
	return values, nil
}

func ParseBytes(data []byte, delimiter rune, hasHeaderRow bool) (Table, error) {
	return ParseReader(bytes.NewReader(data), delimiter, hasHeaderRow)
}

func ParseReader(reader io.Reader, delimiter rune, hasHeaderRow bool) (Table, error) {
	tbl := NewTable()
	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	csvReader.TrimLeadingSpace = true
	idx := -1
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return tbl, err
		}
		idx++
		if idx == 0 && hasHeaderRow {
			tbl.Columns = record
			continue
		}
		tbl.Records = append(tbl.Records, record)
	}
	return tbl, nil
}

// Unmarshal is a convenience function to provide a simple interface to
// unmarshal table contents into any desired output.
func (tbl *Table) Unmarshal(funcRecord func(record []string) error) error {
	for i, rec := range tbl.Records {
		err := funcRecord(rec)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Error on Record Index [%d]", i))
		}
	}
	return nil
}
