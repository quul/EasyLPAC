package main

import (
	"syscall"
	"unsafe"
)

// WIP

type PCSCInterface struct {
	phContext uintptr
}

const SCARD_S_SUCCESS uint32 = 0

var (
	WinSCard                  = syscall.MustLoadDLL("WinSCard.dll")
	procSCardEstablishContext = WinSCard.MustFindProc("SCardEstablishContext")
	procSCardListReadersA     = WinSCard.MustFindProc("SCardListReadersA")
	procSCardReleaseContext   = WinSCard.MustFindProc("SCardReleaseContext")
)

const (
	SCARD_SCOPE_USER   = 0
	SCARD_SCOPE_SYSTEM = 2
)

func (r *PCSCInterface) SCardEstablishContext() error {
	// https://learn.microsoft.com/en-us/windows/win32/api/winscard/nf-winscard-scardestablishcontext
	r1, _, err := procSCardEstablishContext.Call(uintptr(SCARD_SCOPE_USER),
		uintptr(unsafe.Pointer(nil)),
		uintptr(unsafe.Pointer(nil)),
		r.phContext)
	if uint32(r1) != SCARD_S_SUCCESS {
		return err
	}
	return nil
}

func (r *PCSCInterface) SCardReleaseContext() error {
	// https://learn.microsoft.com/en-us/windows/win32/api/winscard/nf-winscard-scardreleasecontext
	r1, _, err := procSCardReleaseContext.Call(
		r.phContext)
	if uint32(r1) != SCARD_S_SUCCESS {
		return err
	}
	return nil
}

func (r *PCSCInterface) SCardListReadersA() ([]string, error) {
	// https://learn.microsoft.com/en-us/windows/win32/api/winscard/nf-winscard-scardlistreadersa
	var mszGroups uintptr = uintptr(unsafe.Pointer(nil))  // 传入 nil 表示获取所有群组的读卡器
	var mszReaders uintptr = uintptr(unsafe.Pointer(nil)) // 为 nil 时忽略 pcchReaders 缓冲区长度，将所需缓冲区长度写入 pcchReaders
	var pcchReaders uint32                                // 缓冲区长度

	// SCardListReadersA 需要进行两次调用，第一次获取所需缓冲区大小，第二次获取读卡器列表

	// 获取所需的缓冲区大小
	r1, _, err := procSCardListReadersA.Call(r.phContext, mszGroups, mszReaders, uintptr(unsafe.Pointer(&pcchReaders)))
	if uint32(r1) != SCARD_S_SUCCESS {
		return nil, err
	}

	// 分配缓冲区
	buffer := make([]byte, pcchReaders)
	mszReaders = uintptr(unsafe.Pointer(&buffer[0]))

	// 第二次调用获取读卡器列表
	r1, _, err = procSCardListReadersA.Call(r.phContext, mszGroups, mszReaders, uintptr(unsafe.Pointer(&pcchReaders)))
	if uint32(r1) != SCARD_S_SUCCESS {
		return nil, err
	}

	// 转换读卡器列表从 null 分隔的字符串到 Go 字符串切片
	readersSlice := (*[1 << 20]byte)(unsafe.Pointer(mszReaders))[:]

	// 用于存储读卡器名称的切片
	var readers []string

	// 从切片中读取读卡器名称
	start := 0
	for i, b := range readersSlice {
		if b == 0 { // 找到 null 字符
			if i == start { // 连续的 null 字符表示列表结束
				break
			}
			// 将非 null 字符部分转换为字符串
			readers = append(readers, string(readersSlice[start:i]))
			start = i + 1
		}
	}
	return readers, nil
}

func NewPCSCInterface() *PCSCInterface {
	return &PCSCInterface{
		phContext: 0,
	}
}
