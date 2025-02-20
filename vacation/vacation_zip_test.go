package vacation_test

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/vacation"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testVacationZip(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
	)

	context("ZipArchive.Decompress", func() {
		var (
			tempDir    string
			zipArchive vacation.ZipArchive
		)

		it.Before(func() {
			var err error
			tempDir, err = os.MkdirTemp("", "vacation")
			Expect(err).NotTo(HaveOccurred())

			buffer := bytes.NewBuffer(nil)
			zw := zip.NewWriter(buffer)

			fileHeader := &zip.FileHeader{Name: "symlink"}
			fileHeader.SetMode(0755 | os.ModeSymlink)

			symlink, err := zw.CreateHeader(fileHeader)
			Expect(err).NotTo(HaveOccurred())

			_, err = symlink.Write([]byte(filepath.Join("some-dir", "some-other-dir", "some-file")))
			Expect(err).NotTo(HaveOccurred())

			// Some archive files will make a relative top level path directory these
			// should still successfully decompress.
			_, err = zw.Create("./")
			Expect(err).NotTo(HaveOccurred())

			_, err = zw.Create("some-dir/")
			Expect(err).NotTo(HaveOccurred())

			_, err = zw.Create(fmt.Sprintf("%s/", filepath.Join("some-dir", "some-other-dir")))
			Expect(err).NotTo(HaveOccurred())

			fileHeader = &zip.FileHeader{Name: filepath.Join("some-dir", "some-other-dir", "some-file")}
			fileHeader.SetMode(0644)

			nestedFile, err := zw.CreateHeader(fileHeader)
			Expect(err).NotTo(HaveOccurred())

			_, err = nestedFile.Write([]byte("nested file"))
			Expect(err).NotTo(HaveOccurred())

			for _, name := range []string{"first", "second", "third"} {
				fileHeader := &zip.FileHeader{Name: name}
				fileHeader.SetMode(0755)

				f, err := zw.CreateHeader(fileHeader)
				Expect(err).NotTo(HaveOccurred())

				_, err = f.Write([]byte(name))
				Expect(err).NotTo(HaveOccurred())
			}

			Expect(zw.Close()).To(Succeed())

			zipArchive = vacation.NewZipArchive(bytes.NewReader(buffer.Bytes()))
		})

		it.After(func() {
			Expect(os.RemoveAll(tempDir)).To(Succeed())
		})

		it("unpackages the archive into the path", func() {
			var err error
			err = zipArchive.Decompress(tempDir)
			Expect(err).ToNot(HaveOccurred())

			files, err := filepath.Glob(fmt.Sprintf("%s/*", tempDir))
			Expect(err).NotTo(HaveOccurred())
			Expect(files).To(ConsistOf([]string{
				filepath.Join(tempDir, "first"),
				filepath.Join(tempDir, "second"),
				filepath.Join(tempDir, "third"),
				filepath.Join(tempDir, "some-dir"),
				filepath.Join(tempDir, "symlink"),
			}))

			info, err := os.Stat(filepath.Join(tempDir, "first"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode()).To(Equal(os.FileMode(0755)))

			Expect(filepath.Join(tempDir, "some-dir", "some-other-dir")).To(BeADirectory())
			Expect(filepath.Join(tempDir, "some-dir", "some-other-dir", "some-file")).To(BeARegularFile())

			link, err := os.Readlink(filepath.Join(tempDir, "symlink"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal("some-dir/some-other-dir/some-file"))

			data, err := os.ReadFile(filepath.Join(tempDir, "symlink"))
			Expect(err).NotTo(HaveOccurred())
			Expect(data).To(Equal([]byte("nested file")))
		})

		context("failure cases", func() {
			context("when it fails to create a zip reader", func() {
				it("returns an error", func() {
					readyArchive := vacation.NewZipArchive(bytes.NewBuffer([]byte(`something`)))

					err := readyArchive.Decompress(tempDir)
					Expect(err).To(MatchError(ContainSubstring("failed to create zip reader")))
				})
			})

			context("when a file is not inside of the destination director (Zip Slip)", func() {
				var buffer *bytes.Buffer
				it.Before(func() {
					var err error
					buffer = bytes.NewBuffer(nil)
					zw := zip.NewWriter(buffer)

					_, err = zw.Create(filepath.Join("..", "some-dir"))
					Expect(err).NotTo(HaveOccurred())

					Expect(zw.Close()).To(Succeed())
				})

				it("returns an error", func() {
					readyArchive := vacation.NewZipArchive(buffer)

					err := readyArchive.Decompress(tempDir)
					Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("illegal file path %q: the file path does not occur within the destination directory", filepath.Join("..", "some-dir")))))
				})

			})

			context("when it fails to unzip a directory", func() {
				var buffer *bytes.Buffer
				it.Before(func() {
					var err error
					buffer = bytes.NewBuffer(nil)
					zw := zip.NewWriter(buffer)

					_, err = zw.Create("some-dir/")
					Expect(err).NotTo(HaveOccurred())

					Expect(zw.Close()).To(Succeed())

					Expect(os.Chmod(tempDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(tempDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					readyArchive := vacation.NewZipArchive(buffer)

					err := readyArchive.Decompress(tempDir)
					Expect(err).To(MatchError(ContainSubstring("failed to unzip directory")))
				})
			})

			context("when it fails to unzip a directory that is part of a file base", func() {
				var buffer *bytes.Buffer
				it.Before(func() {
					var err error
					buffer = bytes.NewBuffer(nil)
					zw := zip.NewWriter(buffer)

					_, err = zw.Create("some-dir/some-file")
					Expect(err).NotTo(HaveOccurred())

					Expect(zw.Close()).To(Succeed())

					Expect(os.Chmod(tempDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(tempDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					readyArchive := vacation.NewZipArchive(buffer)

					err := readyArchive.Decompress(tempDir)
					Expect(err).To(MatchError(ContainSubstring("failed to unzip directory that was part of file path")))
				})
			})

			context("when it tries to symlink to a file that does not exist", func() {
				var buffer *bytes.Buffer
				it.Before(func() {
					var err error
					buffer = bytes.NewBuffer(nil)
					zw := zip.NewWriter(buffer)

					header := &zip.FileHeader{Name: "symlink"}
					header.SetMode(0755 | os.ModeSymlink)

					symlink, err := zw.CreateHeader(header)
					Expect(err).NotTo(HaveOccurred())

					_, err = symlink.Write([]byte(filepath.Join("..", "some-file")))
					Expect(err).NotTo(HaveOccurred())

					Expect(zw.Close()).To(Succeed())

				})

				it("returns an error", func() {
					readyArchive := vacation.NewZipArchive(buffer)

					err := readyArchive.Decompress(tempDir)
					Expect(err).To(MatchError(ContainSubstring("failed to evaluate symlink")))
					Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
				})
			})

			context("when the symlink creation fails", func() {
				var buffer *bytes.Buffer
				it.Before(func() {
					var err error
					buffer = bytes.NewBuffer(nil)
					zw := zip.NewWriter(buffer)

					header := &zip.FileHeader{Name: "symlink"}
					header.SetMode(0755 | os.ModeSymlink)

					symlink, err := zw.CreateHeader(header)
					Expect(err).NotTo(HaveOccurred())

					_, err = symlink.Write([]byte(filepath.Join("some-file")))
					Expect(err).NotTo(HaveOccurred())

					Expect(zw.Close()).To(Succeed())

					// Create a symlink in the target to force the new symlink create to
					// fail
					Expect(os.WriteFile(filepath.Join(tempDir, "some-file"), nil, 0644)).To(Succeed())
					Expect(os.Symlink("some-file", filepath.Join(tempDir, "symlink"))).To(Succeed())
				})

				it("returns an error", func() {
					readyArchive := vacation.NewZipArchive(buffer)

					err := readyArchive.Decompress(tempDir)
					Expect(err).To(MatchError(ContainSubstring("failed to unzip symlink")))
				})
			})

			context("when it fails to unzip a file", func() {
				var buffer *bytes.Buffer
				it.Before(func() {
					var err error
					buffer = bytes.NewBuffer(nil)
					zw := zip.NewWriter(buffer)

					_, err = zw.Create("some-file")
					Expect(err).NotTo(HaveOccurred())

					Expect(zw.Close()).To(Succeed())

					Expect(os.Chmod(tempDir, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(tempDir, os.ModePerm)).To(Succeed())
				})

				it("returns an error", func() {
					readyArchive := vacation.NewZipArchive(buffer)

					err := readyArchive.Decompress(tempDir)
					Expect(err).To(MatchError(ContainSubstring("failed to unzip file")))
				})
			})
		})
	})
}
