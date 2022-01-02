package utils

func RemoveItem(slice []string, item string) []string {
	index := -1
	for i, subject := range slice {
		if subject == item {
			index = i
		}
	}
	if index >= -1 {
		copy(slice[index:], slice[index+1:])
		return slice[:len(slice)-1]
	} else {
		return slice
	}
}
