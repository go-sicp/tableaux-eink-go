// Rockchip EBC userspace bindings, ported verbatim from tableaux-eink/eink-lib.
//
// The exact layout of struct ebc_buf_info and the ioctl numbers MUST be
// verified against the firmware running on the target display before this
// is trusted. Use:
//
//   adb shell strace -f -e trace=ioctl -p $(pidof epdapp)
//
// to confirm.

#ifndef EBC_IOCTL_H
#define EBC_IOCTL_H

#include <linux/ioctl.h>
#include <stdint.h>

#define EBC_IOCTL_MAGIC 'E'

enum ebc_epd_mode {
    EBC_EPD_INIT = 0,
    EBC_EPD_DU = 1,
    EBC_EPD_GC16 = 2,
    EBC_EPD_GCC16 = 3,
    EBC_EPD_A2 = 4,
    EBC_EPD_PART = 5,
    EBC_EPD_FULL = 6,
    EBC_EPD_AUTO = 7,
    EBC_EPD_RESET = 8,
    EBC_EPD_BLACK_WHITE = 9,
    EBC_EPD_TRANSPARENT = 10,
    EBC_EPD_REGAL = 11,
    EBC_EPD_FULL_GC16 = 12,
    EBC_EPD_FULL_GLR16 = 13,
    EBC_EPD_FULL_GLD16 = 14,
    EBC_EPD_FULL_GCC16 = 15,
};

struct ebc_buf_info {
    int32_t width;
    int32_t height;
    int32_t vir_width;
    int32_t vir_height;
    int32_t panel_color;
    int32_t win_x1;
    int32_t win_y1;
    int32_t win_x2;
    int32_t win_y2;
    int32_t epd_mode;
    int32_t buf_fd;
    int32_t needpic;
    int64_t buffer;
};

#define EBC_GET_BUFFER       _IOR(EBC_IOCTL_MAGIC, 0x00, struct ebc_buf_info)
#define EBC_SEND_BUFFER      _IOW(EBC_IOCTL_MAGIC, 0x01, struct ebc_buf_info)
#define EBC_GET_BUFFER_INFO  _IOR(EBC_IOCTL_MAGIC, 0x02, struct ebc_buf_info)
#define EBC_GET_OSD_BUFFER   _IOR(EBC_IOCTL_MAGIC, 0x03, struct ebc_buf_info)
#define EBC_SEND_OSD_BUFFER  _IOW(EBC_IOCTL_MAGIC, 0x04, struct ebc_buf_info)
#define EBC_NEW_BUF_PREPARE  _IOW(EBC_IOCTL_MAGIC, 0x05, struct ebc_buf_info)
#define EBC_ENABLE_OVERLAY   _IO(EBC_IOCTL_MAGIC, 0x06)
#define EBC_DISABLE_OVERLAY  _IO(EBC_IOCTL_MAGIC, 0x07)

#define EBC_DEFAULT_DEV_PATH "/dev/ebc"

#endif
