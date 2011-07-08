
package amf

import (
    "bytes"
    "encoding/hex"
    "fmt"
    "testing"
)
import amf "./protocol"

func testReadAmf3(t *testing.T, blobStr string, expectedStr string) {
    blob,_ := hex.DecodeString(blobStr)
    reader := bytes.NewBuffer(blob)
    val,err := amf.ReadValueAmf3(reader)
    valStr := fmt.Sprintf("%v", val)

    if valStr != expectedStr {
        t.Errorf("Read result of '%s' didn't match expected '%s' for binary blob %s",
            valStr, expectedStr, blobStr)
    }

    if err != nil {
        t.Errorf("Received error while expecting to unpack '%s': %v", expectedStr, err)
    }

    if reader.Len() != 0 {
        t.Errorf("Leftover bytes (%d) while expecting to unpack '%s'", reader.Len(), expectedStr)
    }
}

func testWriteAmf3(t *testing.T, value interface{}, expectedBlob string) {
    expectedBytes,_ := hex.DecodeString(expectedBlob)
    writer := bytes.NewBuffer(make([]byte,0,1))

    err := amf.WriteValueAmf3(writer, value)

    resultBytes := writer.Bytes()
    if bytes.Compare(expectedBytes, resultBytes) != 0 {
        t.Errorf("Write result of '%x' didn't match expected '%s' for input %v",
            resultBytes, expectedBlob, value)
    }

    if err != nil {
        t.Errorf("Received error while trying to write '%v': %v", value, err)
    }
}

func expectReadErrorAmf3(t *testing.T, blobStr string) {
    blob,_ := hex.DecodeString(blobStr)
    reader := bytes.NewBuffer(blob)
    _,err := amf.ReadValueAmf3(reader)

    if err == nil {
        t.Errorf("Expected error but err == nil, for blob: %s", blobStr)
    }
}

func TestSimpleValues(t *testing.T) {
    testReadAmf3(t, "00", "<nil>")
    testReadAmf3(t, "01", "<nil>")
    testReadAmf3(t, "02", "false")
    testReadAmf3(t, "03", "true")

    testWriteAmf3(t, nil, "01")
    testWriteAmf3(t, false, "02")
    testWriteAmf3(t, true, "03")
}

func TestIntegers(t *testing.T) {
    testReadAmf3(t, "0400", "0")
    testReadAmf3(t, "0401", "1")
    testReadAmf3(t, "0401", "1")
    testReadAmf3(t, "0420", "32")
    testReadAmf3(t, "047f", "127")
    testReadAmf3(t, "048001", "1")
    testReadAmf3(t, "04ff7f", "16383")
    testReadAmf3(t, "04ffffffff", "536870911")

    expectReadErrorAmf3(t, "04")
    expectReadErrorAmf3(t, "0480")
    expectReadErrorAmf3(t, "04ffffff")

    testWriteAmf3(t, 1, "0400")
    testWriteAmf3(t, 123, "0400")
    testWriteAmf3(t, 50000, "0400")
    testWriteAmf3(t, 12345678, "0400")
}

func TestDoubles(t *testing.T) {
    testReadAmf3(t, "050000000000000000", "0")
    testReadAmf3(t, "053fbf7ced916872b0", "0.123")
    testReadAmf3(t, "053ff0000000000000", "1")
    testReadAmf3(t, "053fbc71c53f39d1b3", "0.111111")
    testReadAmf3(t, "0540934a456d5cfaad", "1234.5678")

    expectReadErrorAmf3(t, "05")
    expectReadErrorAmf3(t, "0512341234")

    testWriteAmf3(t, 0.0,       "050000000000000000")
    testWriteAmf3(t, 1.0,       "053ff0000000000000")
    testWriteAmf3(t, 1234.0,    "054093480000000000")
    testWriteAmf3(t, 0.111111,  "053fbc71c53f39d1b3")
    testWriteAmf3(t, 1234.5678, "0540934a456d5cfaad")
}

func TestStrings(t *testing.T) {
    testReadAmf3(t, "0601", "")
    testReadAmf3(t, "060361", "a")
    testReadAmf3(t, "060b48656c6c6f", "Hello")
    testReadAmf3(t, "062b546869732069732061206c6f6e6720737472696e67", "This is a long string")

    expectReadErrorAmf3(t, "06")
    expectReadErrorAmf3(t, "0600")
    expectReadErrorAmf3(t, "0603")
    expectReadErrorAmf3(t, "060765")

    testWriteAmf3(t, "", "0601")
    testWriteAmf3(t, "a", "060361")
    testWriteAmf3(t, "Hello", "060b48656c6c6f")
    testWriteAmf3(t, "This is a long string", "062b546869732069732061206c6f6e6720737472696e67")
}

func TestObjects(t *testing.T) {
}

func TestArrays(t *testing.T) {
}

func TestOther(t *testing.T) {
    expectReadErrorAmf3(t, "ff")
}
