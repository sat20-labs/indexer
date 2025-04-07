package indexer

import (
	"fmt"
	"math"
	"strconv"
	"testing"
)

func TestParsePercentage(t *testing.T) {

	fstr := "0.999"
	f, _ := strconv.ParseFloat(fstr, 32)
	fmt.Printf("%s -> %f\n", fstr, f)
	r := (math.Floor(f))
	fmt.Printf("%f -> %f\n", f, r)
	r = (math.Round(f))
	fmt.Printf("%f -> %f\n", f, r)
	r = (math.Trunc(f))
	fmt.Printf("%f -> %f\n", f, r)

	str := "0.991"
	p, err := getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.999"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.9999"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.99"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "1.99"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.90"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "0.990"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "00.990"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "0.09%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.1%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.99%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "1.99%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "90%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "990%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "99.0%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "9.0%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}
}
