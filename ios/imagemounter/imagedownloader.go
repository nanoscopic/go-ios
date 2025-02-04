package imagemounter

import (
	"fmt"
	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/go-ios/ios/zipconduit"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const repo = "https://github.com/haikieu/xcode-developer-disk-image-all-platforms/raw/master/DiskImages/iPhoneOS.platform/DeviceSupport/%s.zip"
const imagepath = "devimages"
const developerDiskImageDmg = "DeveloperDiskImage.dmg"

func GetFileForVersion(version string) string {
  res, err := http.Head( fmt.Sprintf( repo, version ) )
	if err == nil && res.StatusCode != 404 { return version }
	
	versionParts := strings.Split(version, ".")
	if len( versionParts ) != 3 {
	  return ""
	}
	
	twoPartVersion := versionParts[0] + "." + versionParts[1]
	res, err = http.Head( fmt.Sprintf( repo, twoPartVersion ) )
	if err == nil && res.StatusCode != 404 { return twoPartVersion }
	
	return ""
}

func DownloadImageFor(device ios.DeviceEntry, baseDir string) (string, error) {
	allValues, err := ios.GetValues(device)
	if err != nil {
		return "", err
	}
	version := allValues.Value.ProductVersion
	imageDownloaded, err := validateBaseDirAndLookForImage(baseDir, version)
	if err != nil {
		return "", err
	}
	if imageDownloaded != "" {
		log.Infof("%s already downloaded from https://github.com/haikieu/", imageDownloaded)
		return imageDownloaded, nil
	}

	log.Infof("getting developer image for iOS %s", version)
		
	versionToUse := GetFileForVersion(version)
	if versionToUse == "" {
	  // should actually be error; TODO
	  versionToUse = version
	}
	downloadUrl := fmt.Sprintf(repo, versionToUse)
	log.Infof("downloading from: %s", downloadUrl)
	log.Info("thank you haikieu for making these images available :-)")
	zipFileName := path.Join(baseDir, imagepath, fmt.Sprintf("%s.zip", version))
	err = downloadFile(zipFileName, downloadUrl)
	if err != nil {
		return "", err
	}
	files, size, err := zipconduit.Unzip(zipFileName, path.Join(baseDir, imagepath))
	if err != nil {
		return "", err
	}
	err = os.Remove(zipFileName)
	if err != nil {
		log.Warnf("failed deleting: '%s' with err: %+v", zipFileName, err)
	}
	log.Infof("downloaded: %+v totalbytes: %d", files, size)
	downloadedDmgPath, err := findImage(path.Join(baseDir, imagepath), version)
	if err != nil {
		return "", err
	}
	os.RemoveAll(path.Join(baseDir, imagepath, "__MACOSX"))

	log.Infof("Done extracting: %s", downloadedDmgPath)
	return downloadedDmgPath, nil
}

func findImage(dir string, version string) (string, error) {
	versionParts := strings.Split(version, ".")
	twoPartVersion := ""
	if len( versionParts ) == 3 {
	  twoPartVersion = versionParts[0] + "." + versionParts[1]
	}
  
  imageToFind := fmt.Sprintf("%s/%s", version, developerDiskImageDmg)
  twoPartImageToFind := ""
  if twoPartVersion != "" {
    twoPartImageToFind = fmt.Sprintf("%s/%s", twoPartVersion, developerDiskImageDmg)
  }
	imageWeFound := ""
	twoPartImageWeFound := ""
	err := filepath.Walk(dir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(path, imageToFind) {
				imageWeFound = path
			}
			if twoPartImageToFind != "" && strings.HasSuffix(path, twoPartImageToFind) {
			  twoPartImageWeFound = path
			}
			return nil
		})
	if err != nil {
		return "", err
	}
	if imageWeFound != "" {
		return imageWeFound, nil
	}
	if twoPartImageWeFound != "" {
	  return twoPartImageWeFound, nil
	}
	return "", fmt.Errorf("image not found")
}

func validateBaseDirAndLookForImage(baseDir string, version string) (string, error) {
	images := path.Join(baseDir, imagepath)
	dirHandle, err := os.Open(images)
	defer dirHandle.Close()
	if err != nil {
		err := os.MkdirAll(images, 0777)
		if err != nil {
			return "", err
		}
		return "", nil
	}

	dmgPath, err := findImage(baseDir, version)
	if err != nil {
		return "", nil
	}

	return dmgPath, nil
}

// DownloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
// PS: Taken from golangcode.com
func downloadFile(filepath string, url string) error {

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	return err
}
