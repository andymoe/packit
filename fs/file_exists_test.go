package fs_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/packit/fs"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testFileExists(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		dirPath  string
		filePath string
	)

	context("FileExists", func() {
		it.Before(func() {
			var err error
			dirPath, err = os.MkdirTemp("", "dir")
			Expect(err).NotTo(HaveOccurred())
			filePath = filepath.Join(dirPath, "some-file")
		})

		it.After(func() {
			Expect(os.RemoveAll(dirPath)).To(Succeed())
		})

		context("when we are tesing for a file", func() {
			context("and a file exists", func() {
				it.Before(func() {
					Expect(os.WriteFile(filePath, []byte("hello file"), 0644)).To(Succeed())
				})

				it("returns true", func() {
					Expect(fs.FileExists(filePath)).To(BeTrue())
				})
			})

			context("and a file DOES NOT exists", func() {
				it.Before(func() {
					Expect(os.RemoveAll(dirPath)).To(Succeed())
				})

				it("returns false", func() {
					Expect(fs.FileExists(filePath)).To(BeFalse())
				})
			})

			context("when the file cannot be read", func() {
				it.Before(func() {
					Expect(os.WriteFile(filePath, nil, 0644)).To(Succeed())
					Expect(os.Chmod(dirPath, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(dirPath, os.ModePerm)).To(Succeed())
				})

				it("returns false and an error", func() {
					exits, err := fs.FileExists(filePath)
					Expect(err.Error()).To(ContainSubstring("permission denied"))
					Expect(exits).To(BeFalse())
				})
			})
		})

		context("when we are testing for a directory", func() {
			var subDir = filepath.Join(dirPath, "sub-dir")
			context("and a directory exists", func() {
				it("returns true", func() {
					Expect(os.Mkdir(subDir, os.ModePerm)).To(Succeed())
					Expect(fs.FileExists(dirPath)).To(BeTrue())
				})
			})
			context("when the directory does not exist", func() {
				it.Before(func() {
					Expect(os.RemoveAll(dirPath)).To(Succeed())
				})

				it("returns false", func() {
					Expect(fs.FileExists(dirPath)).To(BeFalse())
				})
			})

			context("when the directory cannot be read", func() {
				it.Before(func() {
					Expect(os.Chmod(dirPath, 0000)).To(Succeed())
				})

				it.After(func() {
					Expect(os.Chmod(dirPath, os.ModePerm)).To(Succeed())
				})

				// With this setup we keep getting nil error which I don't understand.
				// I'm clearly missing something.
				it("returns false and an error", func() {
					exits, err := fs.FileExists(subDir)
					Expect(err.Error()).To(ContainSubstring("<syscall.Errno>0xd"))
					Expect(exits).To(BeFalse())
				})
			})
		})
	})
}
