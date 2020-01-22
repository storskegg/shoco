// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package shoco_test

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/storskegg/shoco"
	"github.com/storskegg/shoco/models"
)

func testCompress(in string, proposed bool) string {
	if proposed {
		return hex.EncodeToString(shoco.ProposedCompress([]byte(in)))
	}

	return hex.EncodeToString(shoco.Compress([]byte(in)))
}

func testDecompress(in string, proposed bool) (string, error) {
	b, err := hex.DecodeString(in)
	if err != nil {
		return "", err
	}

	if proposed {
		out, err := shoco.ProposedDecompress(b)
		return string(out), err
	}

	out, err := shoco.Decompress(b)
	return string(out), err
}

// test cases were generated by running:
//  Array.from(shoco.compress("Übergrößenträger")).map(x => ('00' + x.toString(16)).slice(-2)).join('')
// in the development console on https://ed-von-schleck.github.io/shoco/
var testCases = []struct {
	in, out  string
	proposed bool
}{
	{"", "", false},
	{"test", "c899", false},
	{"shoco", "a26fac", false},
	{"shoco is a C library to compress and decompress short strings. It is very fast and easy to use. The default compression model is optimized for english words, but you can generate your own compression model based on your specific input data.", "a26fac20892061204320a6df9b79209120d625ce1d20846420e70484a4737320d09a7420d07199732e2049742089207680792066867420846420658679209120ab652e20549420b86661aa7420d625ce1d698d20b6b86c2089206f70c8db7a8220668e20c04e896820d917732c20bf7420798c20af6e20e908906620798c72206f776e20d625ce1d698d20b6b86c20df5064208d20798c72207370656369666963208870a920dccc2e", false},
	{"shoco is free software, distributed under the MIT license.", "a26fac208920669c6520d11fd8182c20dc499ddeca6420d50072209065204d495420d2b16ea02e", false},
	{"Übergrößenträger", "00c3009cbc72677200c300b600c3009fc05e00c300a46780", false},
	{"Hello, 世界", "48c14d2c2000e400b8009600e70095008c", false},
	{"Go is an open source programming language that makes it easy to build simple, reliable, and efficient software.", "476f20892084206f708120d100ad20709e679f6ddac120d3817561676520c80920b56b83208a20658679209120bf696c6420d0dda42c20ce2a61bd652c20846420656666696369817420d11fd8182e", false},
	{"\u263a\u263b\u2639", "00e2009800ba00e2009800bb00e2009800b9", false},
	{"a\u263ab\u263bc\u2639d", "6100e2009800ba6200e2009800bb6300e2009800b964", false},
	{"1\u20002\u20013\u20024", "3100e2008000803200e2008000813300e20080008234", false},
	{"\u0250\u0250\u0250\u0250\u0250", "00c9009000c9009000c9009000c9009000c90090", false},
	{"\t\v\r\f\n\u0085\u00a0\u2000\u3000", "090b0d0c0a00c2008500c200a000e20080008000e300800080", false},
	{"abcçdefgğhıijklmnoöprsştuüvyz", "61626300c300a7b8666700c4009f6800c400b1696a6b6c6d6e6f00c300b670727300c5009f747500c300bc76797a", false},
	{"ÿøû", "00c300bf00c300b800c300bb", false},
	{"μ", "00ce00bc", false},
	{"μδ", "00ce00bc00ce00b4", false},
	{"\U0001f601", "00f0009f00980081", false},
	{"test\x00test", "c8990000c899", false},

	// See https://github.com/Ed-von-Schleck/shoco/issues/11
	{"μ", "01cebc", true},
	{"μδ", "03cebcceb4", true},
	{"\U0001f601", "03f09f9881", true},
}

func TestCompress(t *testing.T) {
	for i, testCase := range testCases {
		if out := testCompress(testCase.in, testCase.proposed); out != testCase.out {
			t.Errorf("failed for test case #%d", i)
			t.Logf("got:      %s", out)
			t.Logf("expected: %s", testCase.out)
		}
	}
}

func TestDecompress(t *testing.T) {
	for i, testCase := range testCases {
		in, err := testDecompress(testCase.out, testCase.proposed)
		if err != nil {
			t.Errorf("failed for test case #%d", i)
			t.Log(err)
		} else if in != testCase.in {
			t.Errorf("failed for test case #%d", i)
			t.Logf("got:      %s", in)
			t.Logf("expected: %s", testCase.in)
		}
	}
}

var testModels = []struct {
	name  string
	model *shoco.Model
}{
	{"WordsEn", models.WordsEn()},
	{"TextEn", models.TextEn()},
	{"FilePath", models.FilePath()},
	{"Emails", models.Emails()},
}

func TestRoundTrip(t *testing.T) {
	for _, m := range testModels {
		t.Run(m.name, func(t *testing.T) {
			if err := quick.CheckEqual(func(in []byte) (out []byte, err error) {
				return in, nil
			}, func(in []byte) (out []byte, err error) {
				if len(in) == 0 {
					return in, nil
				}

				b := m.model.Compress(in)

				if out, err = m.model.Decompress(b); err != nil {
					t.Logf("in:         %x", in)
					t.Logf("compressed: %x", b)
				}

				return
			}, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestProposedRoundTrip(t *testing.T) {
	for _, m := range testModels {
		t.Run(m.name, func(t *testing.T) {
			if err := quick.CheckEqual(func(in []byte) (out []byte, err error) {
				return in, nil
			}, func(in []byte) (out []byte, err error) {
				if len(in) == 0 {
					return in, nil
				}

				b := m.model.ProposedCompress(in)

				if out, err = m.model.ProposedDecompress(b); err != nil {
					t.Logf("in:         %x", in)
					t.Logf("compressed: %x", b)
				}

				return
			}, nil); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestDecompressASCII(t *testing.T) {
	if err := quick.CheckEqual(func(in []byte) (out []byte, err error) {
		return in, nil
	}, shoco.Decompress, &quick.Config{
		Values: func(values []reflect.Value, rand *rand.Rand) {
			in := make([]byte, 1+rand.Intn(128))
			rand.Read(in)

			for i := range in {
				in[i] &^= 0x80

				for in[i] == 0 {
					in[i] = ^byte(rand.Intn(0x100)) &^ 0x80
				}
			}

			values[0] = reflect.ValueOf(in)
		},
	}); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkCompress(b *testing.B) {
	for i, testCase := range testCases {
		b.Run(fmt.Sprintf("#%d-%d", i, len(testCase.in)), func(b *testing.B) {
			in := []byte(testCase.in)
			b.SetBytes(int64(len(in)))

			if testCase.proposed {
				for n := 0; n < b.N; n++ {
					var _ = shoco.ProposedCompress(in)
				}
			} else {
				for n := 0; n < b.N; n++ {
					var _ = shoco.Compress(in)
				}
			}
		})
	}
}

func BenchmarkDecompress(b *testing.B) {
	for i, testCase := range testCases {
		out, err := hex.DecodeString(testCase.out)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(fmt.Sprintf("#%d-%d", i, len(out)), func(b *testing.B) {
			b.SetBytes(int64(len(out)))

			if testCase.proposed {
				for n := 0; n < b.N; n++ {
					if _, err = shoco.ProposedDecompress(out); err != nil {
						b.Fatal(err)
					}
				}
			} else {
				for n := 0; n < b.N; n++ {
					if _, err = shoco.Decompress(out); err != nil {
						b.Fatal(err)
					}
				}
			}
		})
	}
}

func BenchmarkWords(b *testing.B) {
	f, err := os.Open("/usr/share/dict/words")
	if err != nil {
		if os.IsNotExist(err) {
			b.Skip("/usr/share/dict/words does not exist")
		}

		b.Fatal(err)
	}

	in, err := ioutil.ReadAll(f)
	f.Close()
	if err != nil {
		b.Fatal(err)
	}

	out := shoco.Compress(in)

	b.Logf("len(in)  = %dB", len(in))
	b.Logf("len(out) = %dB", len(out))
	b.Logf("ratio    = %f%%", float64(len(out))/float64(len(in)))

	b.Run("Compress", func(b *testing.B) {
		b.SetBytes(int64(len(in)))

		for n := 0; n < b.N; n++ {
			var _ = shoco.Compress(in)
		}
	})

	b.Run("Decompress", func(b *testing.B) {
		b.SetBytes(int64(len(out)))

		for n := 0; n < b.N; n++ {
			if _, err = shoco.Decompress(out); err != nil {
				b.Fatal(err)
			}
		}
	})
}
