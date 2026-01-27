package s3

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
)

func GetFailureResponse(err error) FailureResponse {
	return FailureResponse{
		Status:       "Failure",
		ResponseCode: http.StatusInternalServerError,
		ErrorMessage: err.Error(),
	}
}

func GetListFolderSuccessResponse(payload *ListFilesResponse) SuccessResponse {
	return SuccessResponse{
		Status:       "Success",
		ResponseCode: http.StatusOK,
		Data:         payload,
	}
}

func GetSuccessResponse(message string) SuccessResponse {
	return SuccessResponse{
		Status:       "Success",
		ResponseCode: http.StatusOK,
		Data:         message,
	}
}

// GetSuccessResponseWithData returns a success response with any data type
func GetSuccessResponseWithData(data interface{}) SuccessResponse {
	return SuccessResponse{
		Status:       "Success",
		ResponseCode: http.StatusOK,
		Data:         data,
	}
}

// custom function to sort the files by name or last modified
func SortFiles(files []ObjectDetails, c echo.Context) *[]ObjectDetails {
	sortBy := c.QueryParam("sortBy")
	order := c.QueryParam("order")

	if sortBy == "" {
		sortBy = "name"
	}

	if sortBy == "name" {
		if order == "asc" {
			sort.Slice(files, func(i, j int) bool {
				return files[i].Name < files[j].Name
			})
		} else {
			sort.Slice(files, func(i, j int) bool {
				return files[i].Name > files[j].Name
			})
		}
	} else if sortBy == "date" {
		if order == "asc" {
			sort.Slice(files, func(i, j int) bool {
				return files[i].LastModified.Before(files[j].LastModified)
			})
		} else {
			sort.Slice(files, func(i, j int) bool {
				return files[i].LastModified.After(files[j].LastModified)
			})
		}
	} else if sortBy == "type" {
		if order == "asc" {
			sort.Slice(files, func(i, j int) bool {
				return files[i].IsFolder
			})
		} else {
			sort.Slice(files, func(i, j int) bool {
				return !files[i].IsFolder
			})
		}
	} else if sortBy == "size" {
		if order == "asc" {
			sort.Slice(files, func(i, j int) bool {
				return files[i].Size < files[j].Size
			})
		} else {
			sort.Slice(files, func(i, j int) bool {
				return files[i].Size > files[j].Size
			})
		}
	}

	return &files
}

// custom function to filter the files by size range, date range, file type, filename and file size
func FilterFiles(files []ObjectDetails, options FilterOptions) *[]ObjectDetails {
	var filteredFiles []ObjectDetails
	filterFilesBySizeRange := func(sizeRange string, files []ObjectDetails) *[]ObjectDetails {
		rangeValues, found := sizeRanges[sizeRange]
		if !found {
			return nil // Invalid size range
		}

		var filesInRange []ObjectDetails
		for _, file := range files {
			if file.Size >= rangeValues.MinSize && (rangeValues.MaxSize == -1 || file.Size <= rangeValues.MaxSize) {
				filesInRange = append(filesInRange, file)
			}
		}
		return &filesInRange
	}

	filterFilesByTimeRange := func(dateRange string, files []ObjectDetails) *[]ObjectDetails {
		duration, found := timeRanges[dateRange]
		if !found {
			return nil // Invalid date range
		}

		var filesInRange []ObjectDetails
		cutoffTime := time.Now().Add(duration)
		for _, file := range files {
			if file.LastModified.After(cutoffTime) {
				filesInRange = append(filesInRange, file)
			}
		}

		return &filesInRange
	}

	filterFilesByTypes := func(fileTypes []string, files []ObjectDetails) *[]ObjectDetails {
		var filesInRange []ObjectDetails
		for _, file := range files {
			// get the file extension by splitting the file name and picking the last element
			ext := strings.Split(file.Name, ".")[len(strings.Split(file.Name, "."))-1]

			// check if the file extension is present in the fileTypes array
			for _, fileType := range fileTypes {
				if fileType == ext {
					filesInRange = append(filesInRange, file)
				}
			}
		}

		return &filesInRange
	}

	filterFilesByFilename := func(query, filterType string, files []ObjectDetails) *[]ObjectDetails {
		var filesInRange []ObjectDetails

		for _, file := range files {
			filename := strings.ToLower(file.Name)

			switch filterType {
			case "contains":
				if strings.Contains(filename, strings.ToLower(query)) {
					filesInRange = append(filesInRange, file)
				}
			case "startsWith":
				if strings.HasPrefix(filename, strings.ToLower(query)) {
					filesInRange = append(filesInRange, file)
				}
			case "endsWith":
				if strings.HasSuffix(filename, strings.ToLower(query)) {
					filesInRange = append(filesInRange, file)
				}
			}
		}

		return &filesInRange
	}

	filterFilesByFileSize := func(fileSize int64, filterType string, files []ObjectDetails) *[]ObjectDetails {
		var filesInRange []ObjectDetails

		for _, file := range files {
			switch filterType {
			case "gt":
				if file.Size >= fileSize {
					filesInRange = append(filesInRange, file)
				}
			case "gte":
				if file.Size > fileSize {
					filesInRange = append(filesInRange, file)
				}
			case "lt":
				if file.Size < fileSize {
					filesInRange = append(filesInRange, file)
				}
			case "lte":
				if file.Size <= fileSize {
					filesInRange = append(filesInRange, file)
				}
			case "eq":
				if file.Size == fileSize {
					filesInRange = append(filesInRange, file)
				}
			}
		}

		return &filesInRange
	}

	// Filter by size range
	if options.SizeRange != "" {
		filteredFiles = *filterFilesBySizeRange(options.SizeRange, files)
	} else {
		filteredFiles = files
	}

	// Filter by date range
	if options.TimeRange != "" {
		filteredFiles = *filterFilesByTimeRange(options.TimeRange, filteredFiles)
	}

	// Filter by file type
	if options.FileTypes != nil {
		filteredFiles = *filterFilesByTypes(options.FileTypes, filteredFiles)
	}

	// Filter by filename
	if options.FilenameQuery != "" && options.FilenameFilterType != "" {
		filteredFiles = *filterFilesByFilename(options.FilenameQuery, options.FilenameFilterType, filteredFiles)
	}

	// Filter by file size
	if options.FileSizeFilterType != "" {
		filteredFiles = *filterFilesByFileSize(options.FileSize, options.FileSizeFilterType, filteredFiles)
	}

	return &filteredFiles
}
