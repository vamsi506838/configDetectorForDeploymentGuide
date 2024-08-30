package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {

	scanner := bufio.NewScanner(os.Stdin)

	// Prompt the user for input
	fmt.Print("Please paste the helmchart link for any of the component (which should be in higher version) : ")

	// Read the user's input
	scanner.Scan()
	srcurl := scanner.Text()

	// URL of the .tgz file
	urlSRC := strings.TrimSpace(srcurl) //srcurl //"https://helmcharts.qstack.com/testing/cloud-volumes-network-2401.0.0-RC.9.tgz"
	srcMap, err := DownloadFolderOne(urlSRC)
	if err != nil {
		fmt.Println("extraction failed")
	}

	fmt.Print("Please paste the helmchart link for any of the component (which should be in lower version): ")

	// Read the user's input
	scanner.Scan()
	dsturl := scanner.Text()

	// URL of the .tgz file
	urlDST := strings.TrimSpace(dsturl) //"https://helmcharts.qstack.com/testing/cloud-volumes-network-2403.0.0-DEV.9.tgz"
	dstMap, err := DownloadFolderOne(urlDST)
	if err != nil {
		fmt.Println("extraction failed")
	}

	fmt.Println("Difference between current and previous-> added ")
	fmt.Println()

	for key, _ := range dstMap {

		if strings.Contains(key, "values.yaml") { //strings.Contains(key, "deployment.yaml") || strings.Contains(key, "values.yaml") {

			//break
			fmt.Println()
			title := key
			title = strings.TrimSuffix(title, title[len(title)-12:])
			//fmt.Println("-->" + title + ":")
			//fmt.Println()
			resSrc := printAddedOne(key, srcMap, dstMap)

			if len(resSrc) > 0 {
				fmt.Println()
				fmt.Println("-->" + title + ":")
				for _, v := range resSrc {
					if strings.Contains(v, ".tag:") || strings.Contains(v, ".digest:") {
						continue
					}
					reset := "\033[0m"
					green := "\033[32m"
					markedText := fmt.Sprintf("%s%s%s", green, v, reset)
					fmt.Println(markedText)
				}
			}
		}
	}

	fmt.Println("Difference between previous and current ->  removed ")
	fmt.Println()

	for key, _ := range srcMap {

		if strings.Contains(key, "values.yaml") { //strings.Contains(key, "deployment.yaml") || strings.Contains(key, "values.yaml") {

			fmt.Println()
			title := key
			title = strings.TrimSuffix(title, title[len(title)-12:])
			//fmt.Println("-->" + title + ":")
			//fmt.Println()

			//break

			resSrc := printRemovedOne(key, srcMap, dstMap)
			//fmt.Println("length->", len(resSrc))
			if len(resSrc) > 0 {
				//fmt.Println()
				fmt.Println("-->" + title + ":")
				for _, v := range resSrc {
					if strings.Contains(v, ".tag:") || strings.Contains(v, ".digest:") {
						continue
					}
					reset := "\033[0m"
					red := "\033[31m"
					markedText := fmt.Sprintf("%s%s%s", red, v, reset)
					fmt.Println(markedText)
				}
			}

		}
	}

}

func printAddedOne(str1 string, srcMap map[string]string, dstMap map[string]string) []string {

	val1 := srcMap[str1]
	val2 := dstMap[str1]

	/*lines1 := strings.Split(val1, "\n")
	lines2 := strings.Split(val2, "\n")

	distinctLines := make(map[string]bool)

	for _, line := range lines2 {
		distinctLines[line] = true
	}

	for _, line := range lines1 {
		if _, exists := distinctLines[line]; exists {
			delete(distinctLines, line)
		} else {
			distinctLines[line] = false
		}
	}

	for line, isDistinct := range distinctLines {
		if isDistinct {
			if strings.Contains(line, "digest") || strings.Contains(line, "tag") {
				continue
			}
			fmt.Println(line)
		}
	}*/

	convertedYAMLSRC := convertYAML(val1)
	convertedYAMLDST := convertYAML(val2)

	stringArrSrc := strings.Split(convertedYAMLSRC, "\n")
	stringArrDst := strings.Split(convertedYAMLDST, "\n")

	var finalOutput string
	for _, keySrcValue := range stringArrSrc {
		flag := false
		for _, keyDstValue := range stringArrDst {
			if keySrcValue == keyDstValue {
				flag = true
				break
			}
		}
		if flag == false {
			finalOutput = finalOutput + keySrcValue + "<->"
		}
	}

	//fmt.Println(finalOutput)
	//fmt.Println(convertedYAML)
	var res []string
	strArr := strings.Split(finalOutput, "<->")
	for _, v := range strArr {
		res = append(res, v)

		/*if !strings.Contains(v, ".tag:") || !strings.Contains(v, ".digest:") {
		reset := "\033[0m"
		green := "\033[32m"
		markedText := fmt.Sprintf("%s%s%s", green, v, reset)
		fmt.Println(markedText)
		}*/
	}

	return res
}

func convertYAML(yamlStr string) string {
	// Parse the YAML string into a map
	var data map[string]interface{}
	yaml.Unmarshal([]byte(yamlStr), &data)

	// Recursively traverse the map and create the desired format
	var convertedData []string
	traverseMap(data, "", &convertedData)

	// Join the converted data with newlines
	convertedYAML := strings.Join(convertedData, "\n")
	return convertedYAML
}

func traverseMap(data map[string]interface{}, prefix string, convertedData *[]string) {
	for key, value := range data {
		newKey := fmt.Sprintf("%s.%s", prefix, key)
		if prefix == "" {
			newKey = key
		}

		if nestedMap, ok := value.(map[interface{}]interface{}); ok {
			traverseMap(convertInterfaceMapToStringMap(nestedMap), newKey, convertedData)
		} else {
			convertedDataLine := fmt.Sprintf("%s: %v", newKey, value)
			*convertedData = append(*convertedData, convertedDataLine)
		}
	}
}

func convertInterfaceMapToStringMap(input map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for key, value := range input {
		output[fmt.Sprintf("%v", key)] = value
	}
	return output
}

func printRemovedOne(str1 string, srcMap map[string]string, dstMap map[string]string) []string {

	val1 := srcMap[str1]
	val2 := dstMap[str1]

	/*lines1 := strings.Split(val1, "\n")
	lines2 := strings.Split(val2, "\n")

	distinctLines := make(map[string]bool)

	for _, line := range lines1 {
		distinctLines[line] = true
	}

	for _, line := range lines2 {
		if _, exists := distinctLines[line]; exists {
			delete(distinctLines, line)
		} else {
			distinctLines[line] = false
		}
	}

	for line, isDistinct := range distinctLines {
		if isDistinct {
			if strings.Contains(line, "digest") || strings.Contains(line, "tag") {
				continue
			}
			fmt.Println(line)
		}
	}*/
	convertedYAMLSRC := convertYAML(val2)
	convertedYAMLDST := convertYAML(val1)

	stringArrSrc := strings.Split(convertedYAMLSRC, "\n")
	stringArrDst := strings.Split(convertedYAMLDST, "\n")

	var finalOutput string
	for _, keySrcValue := range stringArrSrc {
		flag := false
		for _, keyDstValue := range stringArrDst {
			if keySrcValue == keyDstValue {
				flag = true
				break
			}
		}
		if flag == false {
			finalOutput = finalOutput + keySrcValue + "<->"
		}
	}

	//fmt.Println(finalOutput)
	var res []string
	strArr := strings.Split(finalOutput, "<->")
	for _, v := range strArr {
		res = append(res, v)
		/*if !strings.Contains(v, ".tag:") || !strings.Contains(v, ".digest:") {
			reset := "\033[0m"
			red := "\033[31m"
			markedText := fmt.Sprintf("%s%s%s", red, v, reset)
			fmt.Println(markedText)
		}*/
	}
	return res
}

func DownloadFolderOne(URL string) (map[string]string, error) {

	response, err := http.Get(URL)
	if err != nil {
		fmt.Println("Failed to download .tgz file:", err)
		return nil, err
	}
	defer response.Body.Close()

	// Create the .tgz file
	file, err := os.Create("yourfile.tgz")
	if err != nil {
		fmt.Println("Failed to create .tgz file:", err)
		return nil, err
	}
	defer file.Close()

	// Copy the contents of the downloaded file to the .tgz file
	_, err = io.Copy(file, response.Body)
	if err != nil {
		fmt.Println("Failed to save .tgz file:", err)
		return nil, err
	}

	// Open the .tgz file
	file, err = os.Open("yourfile.tgz")
	if err != nil {
		fmt.Println("Failed to open .tgz file:", err)
		return nil, err
	}
	defer file.Close()

	// Create a gzip reader
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		fmt.Println("Failed to create gzip reader:", err)
		return nil, err
	}
	defer gzipReader.Close()

	// Create a tar reader
	tarReader := tar.NewReader(gzipReader)

	// Map to store file data
	fileData := make(map[string]string)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println("Failed to read tar header:", err)
			return nil, err
		}

		data, err := io.ReadAll(tarReader)
		if err != nil {
			fmt.Println("Failed to read file contents:", err)
			return nil, err
		}

		fileData[header.Name] = string(data)
	}

	return fileData, nil
}

/*

package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"fmt"
	"gopkg.in/yaml.v2"
	"io"
	"net/http"
	"os"
	"strings"
)

func main() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Print("Please paste the helmchart link for any of the component (which should be in higher version): ")
	scanner.Scan()
	srcURL := strings.TrimSpace(scanner.Text())

	srcMap, err := downloadAndExtract(srcURL)
	if err != nil {
		fmt.Println("Extraction failed:", err)
		return
	}

	fmt.Print("Please paste the helmchart link for any of the component (which should be in lower version): ")
	scanner.Scan()
	dstURL := strings.TrimSpace(scanner.Text())

	dstMap, err := downloadAndExtract(dstURL)
	if err != nil {
		fmt.Println("Extraction failed:", err)
		return
	}

	printDifferences("Difference between current and previous -> added", srcMap, dstMap, printAdded)
	printDifferences("Difference between previous and current -> removed", dstMap, srcMap, printRemoved)
}

func printDifferences(title string, srcMap, dstMap map[string]string, diffFunc func(string, map[string]string, map[string]string) []string) {
	fmt.Println(title)
	for key := range srcMap {
		if strings.Contains(key, "values.yaml") {
			fmt.Println()
			fmt.Println("-->", strings.TrimSuffix(key, key[len(key)-12:]), ":")
			for _, line := range diffFunc(key, srcMap, dstMap) {
				if !strings.Contains(line, ".tag:") && !strings.Contains(line, ".digest:") {
					fmt.Println(line)
				}
			}
		}
	}
}

func printAdded(key string, srcMap, dstMap map[string]string) []string {
	return diffYAML(srcMap[key], dstMap[key])
}

func printRemoved(key string, srcMap, dstMap map[string]string) []string {
	return diffYAML(dstMap[key], srcMap[key])
}

func diffYAML(src, dst string) []string {
	srcLines := strings.Split(convertYAML(src), "\n")
	dstLines := strings.Split(convertYAML(dst), "\n")
	var diff []string
	for _, srcLine := range srcLines {
		found := false
		for _, dstLine := range dstLines {
			if srcLine == dstLine {
				found = true
				break
			}
		}
		if !found {
			diff = append(diff, srcLine)
		}
	}
	return diff
}

func convertYAML(yamlStr string) string {
	var data map[string]interface{}
	yaml.Unmarshal([]byte(yamlStr), &data)
	var convertedData []string
	traverseMap(data, "", &convertedData)
	return strings.Join(convertedData, "\n")
}

func traverseMap(data map[string]interface{}, prefix string, convertedData *[]string) {
	for key, value := range data {
		newKey := key
		if prefix != "" {
			newKey = fmt.Sprintf("%s.%s", prefix, key)
		}
		if nestedMap, ok := value.(map[interface{}]interface{}); ok {
			traverseMap(convertInterfaceMapToStringMap(nestedMap), newKey, convertedData)
		} else {
			*convertedData = append(*convertedData, fmt.Sprintf("%s: %v", newKey, value))
		}
	}
}

func convertInterfaceMapToStringMap(input map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{})
	for key, value := range input {
		output[fmt.Sprintf("%v", key)] = value
	}
	return output
}

func downloadAndExtract(URL string) (map[string]string, error) {
	response, err := http.Get(URL)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	file, err := os.CreateTemp("", "*.tgz")
	if err != nil {
		return nil, err
	}
	defer os.Remove(file.Name())

	if _, err := io.Copy(file, response.Body); err != nil {
		return nil, err
	}

	if _, err := file.Seek(0, 0); err != nil {
		return nil, err
	}

	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, err
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	fileData := make(map[string]string)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(tarReader)
		if err != nil {
			return nil, err
		}
		fileData[header.Name] = string(data)
	}
	return fileData, nil
}
*/
