package service

import (
	"archive/zip"
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanService_InspectFile_SingleDLL(t *testing.T) {
	scanner := NewScanService("")
	dummyDLL := []byte("MZ\x90\x00\x03\x00\x00\x00dummy-dll-data")

	res, err := scanner.InspectFile("MyMod.dll", dummyDLL)
	require.NoError(t, err)
	assert.NotEmpty(t, res.SHA256)
	assert.NotEmpty(t, res.MD5)
	assert.True(t, res.ContainsDLL)
	assert.Equal(t, []string{"MyMod.dll"}, res.DLLFiles)
	assert.Empty(t, res.SuspiciousFiles)
	assert.Empty(t, res.Warnings)
}

func TestScanService_InspectFile_ZipWithDLL(t *testing.T) {
	scanner := NewScanService("")

	// Create in-memory zip
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	f1, err := zipWriter.Create("plugins/SuperMod.dll")
	require.NoError(t, err)
	_, _ = f1.Write([]byte("fake dll content"))

	f2, err := zipWriter.Create("README.txt")
	require.NoError(t, err)
	_, _ = f2.Write([]byte("Instructions"))

	err = zipWriter.Close()
	require.NoError(t, err)

	res, err := scanner.InspectFile("SuperMod.zip", buf.Bytes())
	require.NoError(t, err)
	assert.True(t, res.IsArchive)
	assert.True(t, res.ContainsDLL)
	assert.Contains(t, res.DLLFiles, "plugins/SuperMod.dll")
	assert.Empty(t, res.SuspiciousFiles)
}

func TestScanService_InspectFile_SuspiciousExecutableInZip(t *testing.T) {
	scanner := NewScanService("")

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	f1, err := zipWriter.Create("malicious.bat")
	require.NoError(t, err)
	_, _ = f1.Write([]byte("echo harmful script"))

	err = zipWriter.Close()
	require.NoError(t, err)

	res, err := scanner.InspectFile("ModWithScript.zip", buf.Bytes())
	require.NoError(t, err)
	assert.NotEmpty(t, res.SuspiciousFiles)
	assert.Contains(t, res.SuspiciousFiles, "malicious.bat")
	assert.NotEmpty(t, res.Warnings)
}

func TestScanService_InspectFile_ZipSlipDetection(t *testing.T) {
	scanner := NewScanService("")

	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	f1, err := zipWriter.Create("../../../windows/system32/evil.dll")
	require.NoError(t, err)
	_, _ = f1.Write([]byte("evil"))

	err = zipWriter.Close()
	require.NoError(t, err)

	res, err := scanner.InspectFile("ZipSlip.zip", buf.Bytes())
	require.NoError(t, err)
	assert.NotEmpty(t, res.SuspiciousFiles)
	assert.Contains(t, res.SuspiciousFiles, "../../../windows/system32/evil.dll")
}
