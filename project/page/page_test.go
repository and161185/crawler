package page

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetTitle(t *testing.T) {
	testString := `<title>TestTitle</title>
		<a href="localhost1">linkText</a>`

	reader := strings.NewReader(testString)

	page, _ := NewPage(reader)
	correctResult := "TestTitle"
	result := page.GetTitle()

	if correctResult != result {
		t.Errorf("Wrong title. Need %s got %s ", correctResult, result)
	}
}

func TestGetLinks(t *testing.T) {

	testString := `<title>TestTitle</title>
		<a href="localhost1">linkText</a>
		<a href="localhost2">linkText</a>`

	reader := strings.NewReader(testString)

	page, _ := NewPage(reader)

	correctResult := []string{"localhost1", "localhost2"}
	result := page.GetLinks()

	assert.Equal(t, correctResult, result)
}
