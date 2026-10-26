//go:build darwin && cgo

package hub

/*
#cgo LDFLAGS: -framework CoreServices
#include <CoreServices/CoreServices.h>
#include <stdlib.h>

static CFStringRef utiFor(const char *ext) {
	CFStringRef e = CFStringCreateWithCString(NULL, ext, kCFStringEncodingUTF8);
	if (e == NULL) return NULL;
	CFStringRef uti = UTTypeCreatePreferredIdentifierForTag(kUTTagClassFilenameExtension, e, NULL);
	CFRelease(e);
	return uti;
}

static OSStatus setDefaultHandler(const char *ext, const char *bundleID) {
	CFStringRef b = CFStringCreateWithCString(NULL, bundleID, kCFStringEncodingUTF8);
	if (b == NULL) return -1;
	CFStringRef uti = utiFor(ext);
	if (uti == NULL) { CFRelease(b); return -2; }
	OSStatus status = LSSetDefaultRoleHandlerForContentType(uti, kLSRolesAll, b);
	CFRelease(uti);
	CFRelease(b);
	return status;
}

// getDefaultHandler writes the current handler's bundle ID into buf ("" if none).
static void getDefaultHandler(const char *ext, char *buf, int n) {
	buf[0] = 0;
	CFStringRef uti = utiFor(ext);
	if (uti == NULL) return;
	CFStringRef h = LSCopyDefaultRoleHandlerForContentType(uti, kLSRolesAll);
	CFRelease(uti);
	if (h == NULL) return;
	if (!CFStringGetCString(h, buf, n, kCFStringEncodingUTF8)) buf[0] = 0;
	CFRelease(h);
}
*/
import "C"

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"unsafe"
)

const lsregister = "/System/Library/Frameworks/CoreServices.framework/Frameworks/LaunchServices.framework/Support/lsregister"

func setDefaultApp(app App, exts []string, in *Installed) error {
	bundle := in.Exe
	id, err := macBundleID(bundle)
	if err != nil {
		return err
	}
	// Teach Launch Services about the freshly installed bundle before asking
	// it to hand our formats to that bundle ID.
	if out, err := exec.Command(lsregister, "-f", bundle).CombinedOutput(); err != nil {
		return fmt.Errorf("register %s with Launch Services: %v: %s", filepath.Base(bundle), err, strings.TrimSpace(string(out)))
	}
	var bad []string
	applied := 0
	for _, ext := range exts {
		cext, cid := C.CString(ext), C.CString(id)
		status := C.setDefaultHandler(cext, cid)
		C.free(unsafe.Pointer(cext))
		C.free(unsafe.Pointer(cid))
		switch status {
		case 0:
			applied++
		case -50: // paramErr: the bundle claims none of this type; skip it
		default:
			bad = append(bad, ext)
		}
	}
	if applied == 0 {
		return fmt.Errorf("%s declares none of its document types to macOS (Info.plist), so it cannot become their default app", app.Name)
	}
	if len(bad) > 0 {
		return fmt.Errorf("macOS rejected the default handler for %s", strings.Join(bad, ", "))
	}
	return nil
}

// currentDefaults returns each extension's current handler bundle ID.
func currentDefaults(app App, exts []string) map[string]string {
	prev := map[string]string{}
	buf := (*C.char)(C.malloc(1024))
	defer C.free(unsafe.Pointer(buf))
	for _, ext := range exts {
		cext := C.CString(ext)
		C.getDefaultHandler(cext, buf, 1024)
		C.free(unsafe.Pointer(cext))
		if id := C.GoString(buf); id != "" {
			prev[ext] = id
		}
	}
	return prev
}

// restoreDefaults hands each extension back to its recorded handler. macOS has
// no "unset", so extensions without a recorded handler keep the Craft app
// until the user picks another in Finder (or the bundle is removed).
func restoreDefaults(app App, exts []string, prev map[string]string) error {
	var bad []string
	for _, ext := range exts {
		id := prev[ext]
		if id == "" {
			continue
		}
		cext, cid := C.CString(ext), C.CString(id)
		status := C.setDefaultHandler(cext, cid)
		C.free(unsafe.Pointer(cext))
		C.free(unsafe.Pointer(cid))
		if status != 0 {
			bad = append(bad, ext)
		}
	}
	if len(bad) > 0 {
		return fmt.Errorf("macOS rejected the previous handler for %s", strings.Join(bad, ", "))
	}
	return nil
}

func macBundleID(bundle string) (string, error) {
	out, err := exec.Command("/usr/bin/plutil", "-extract", "CFBundleIdentifier", "raw", "-o", "-",
		filepath.Join(bundle, "Contents", "Info.plist")).Output()
	id := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("read %s's bundle ID: %w", filepath.Base(bundle), err)
	}
	if id == "" {
		return "", fmt.Errorf("%s has no CFBundleIdentifier", filepath.Base(bundle))
	}
	return id, nil
}
