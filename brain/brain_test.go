package brain

import (
	"fmt"
	"testing"
)

func TestNewBehaviors(t *testing.T) {
	behaviors, err := NewBehaviors("hello")
	fmt.Println(behaviors, err)
}
