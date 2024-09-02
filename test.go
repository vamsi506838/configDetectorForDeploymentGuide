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

	addedMap := make(map[string][]string)
	for key, _ := range dstMap {
		if strings.Contains(key, "values.yaml") {
			title := key
			title = strings.TrimSuffix(title, title[len(title)-12:])
			resSrc := printAddedOne(key, srcMap, dstMap)
			var addedList []string
			if len(resSrc) > 0 {
				for _, v := range resSrc {
					if strings.Contains(v, ".tag:") || strings.Contains(v, ".digest:") {
						continue
					}
					addedList = append(addedList, v)
				}
			}
			addedMap[title] = addedList
		}
	}

	removedMap := make(map[string][]string)
	for key, _ := range srcMap {
		if strings.Contains(key, "values.yaml") { //strings.Contains(key, "deployment.yaml") || strings.Contains(key, "values.yaml") {
			title := key
			title = strings.TrimSuffix(title, title[len(title)-12:])

			var removedList []string
			resSrc := printRemovedOne(key, srcMap, dstMap)
			if len(resSrc) > 0 {
				for _, v := range resSrc {
					if strings.Contains(v, ".tag:") || strings.Contains(v, ".digest:") {
						continue
					}
					removedList = append(removedList, v)
				}
			}
			removedMap[title] = removedList
		}
	}

	fmt.Println("--------------------Final Result----------------")
	updatedMap := updatedMapFunc(addedMap, removedMap)
	for updatedKey, updatedListValue := range updatedMap {
		addedValue := addedMap[updatedKey]
		removedValue := removedMap[updatedKey]
		for _, updatedValue := range updatedListValue {
			// Remove updatedValue from addedValue
			for i, v := range addedValue {
				arrAdded := strings.Split(v, ":")
				arrUpdated := strings.Split(updatedValue, ":")
				if arrAdded[0] == arrUpdated[0] {
					addedValue = append(addedValue[:i], addedValue[i+1:]...)
					break
				}
			}
			// Remove updatedValue from removedValue
			for i, v := range removedValue {
				arr := strings.Split(v, ":")
				arrUpdated := strings.Split(updatedValue, ":")
				if arr[0] == arrUpdated[0] {
					removedValue = append(removedValue[:i], removedValue[i+1:]...)
					break
				}
			}
		}
		addedMap[updatedKey] = addedValue
		removedMap[updatedKey] = removedValue
	}

	fmt.Println("Added Configs:")
	for key, value := range addedMap {
		if len(value) > 0 {
			fmt.Println(key)
			for _, v := range value {
				reset := "\033[0m"
				green := "\033[32m"
				markedText := fmt.Sprintf("%s%s%s", green, v, reset)
				fmt.Println(markedText)
			}
		}
	}
	fmt.Println("Removed Configs:")
	for key, value := range removedMap {
		if len(value) > 0 {
			fmt.Println(key)
			for _, v := range value {
				reset := "\033[0m"
				red := "\033[31m"
				markedText := fmt.Sprintf("%s%s%s", red, v, reset)
				fmt.Println(markedText)
			}
		}
	}
	fmt.Println("Updated Configs:")
	for key, value := range updatedMap {
		if len(value) > 0 {
			fmt.Println(key)
			for _, v := range value {
				reset := "\033[0m"
				yellow := "\033[33m"
				markedText := fmt.Sprintf("%s%s%s", yellow, v, reset)
				fmt.Println(markedText)
			}
		}
	}
}

func updatedMapFunc(addedMap map[string][]string, removedMap map[string][]string) map[string][]string {
	addSrcMap := make(map[string]string)
	removeSrcMap := make(map[string]string)
	updatedMap := make(map[string][]string)
	for addedKey, addedListValue := range addedMap {
		removedListValue, exists := removedMap[addedKey]
		if exists {
			addSrcMap = convertToMap(addedListValue)
			removeSrcMap = convertToMap(removedListValue)
		}
		for addKey, addValue := range addSrcMap {
			updatedList := make([]string, 0)
			for removeKey, removeValue := range removeSrcMap {
				if addKey == removeKey {
					if addValue != removeValue {
						res := addKey + ": " + addValue
						updatedList = append(updatedList, res)
						updatedMap[addedKey] = updatedList
					}
				}
			}
		}
	}
	return updatedMap
}

func convertToMap(lines []string) map[string]string {
	result := make(map[string]string)
	for _, line := range lines {
		parts := strings.Split(line, ":")
		key := strings.TrimSpace(parts[0])
		if len(parts) == 2 {
			value := strings.TrimSpace(parts[1])
			result[key] = value
		}
	}
	return result
}

func printAddedOne(str1 string, srcMap map[string]string, dstMap map[string]string) []string {

	val1 := srcMap[str1]
	val2 := dstMap[str1]

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

	var res []string
	strArr := strings.Split(finalOutput, "<->")
	for _, v := range strArr {
		res = append(res, v)
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
