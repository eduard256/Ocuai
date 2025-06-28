# Camera Integration Guide

## Overview

Ocuai now includes advanced camera integration powered by go2rtc, supporting a wide range of IP cameras including:

- Modern RTSP/ONVIF cameras (Dahua, Hikvision, Axis, etc.)
- TP-Link Tapo cameras
- DVR-IP/XMeye cameras
- Legacy HTTP/MJPEG cameras
- And many more proprietary protocols

## Features

- **Universal Camera Scanner**: Automatically discovers all available streams from any IP camera
- **Smart Stream Selection**: Recommends the best stream based on protocol priority
- **Zero Configuration**: No need to know RTSP URLs or camera specifications
- **Legacy Support**: Works with even the oldest IP cameras
- **Multiple Protocols**: Supports RTSP, RTMP, HTTP, ONVIF, Tapo, DVR-IP, and more
- **Real-time Streaming**: Low-latency WebRTC streaming in the browser
- **Two-way Audio**: Supported for compatible cameras

## Adding a Camera

1. **Click "Add Camera"** in the Cameras page
2. **Enter Camera Details**:
   - IP Address (IPv4 format, e.g., 192.168.1.100)
   - Username (default: admin)
   - Password (leave empty if not required)
   - Camera Name (for display)
3. **Click "Continue"** to start scanning
4. **Select Stream** from the discovered list
5. **Click "Add Camera"** to complete

## Supported Camera Brands

### Tier 1 (Best Support)
- **Dahua**: Full RTSP/ONVIF support with two-way audio
- **Hikvision**: RTSP + ISAPI for two-way audio
- **Axis**: Professional grade streaming
- **TP-Link Tapo**: Native protocol support

### Tier 2 (Good Support)
- **Amcrest**
- **Reolink** (avoid RTSP if possible)
- **Ubiquiti UniFi**
- **Foscam**
- **Generic ONVIF cameras**

### Tier 3 (Basic Support)
- **Chinese no-name cameras**
- **Wyze with hacks**
- **Xiaomi with hacks**
- **Legacy analog-to-IP converters**

## Technical Details

### Scanner Operation

The camera scanner performs an exhaustive search for all possible stream URLs:

1. **RTSP Streams**: Checks common ports (554, 8554, 88, 10554) with various paths
2. **HTTP Streams**: Scans for MJPEG, snapshots, and HLS streams
3. **ONVIF Discovery**: Automatically finds all streams via ONVIF protocol
4. **Proprietary Protocols**: Tests vendor-specific protocols (Tapo, DVR-IP, etc.)

### Stream Validation

Each discovered stream is tested for connectivity before being presented to the user. Only working streams are shown.

### Integration with go2rtc

All camera streams are managed by go2rtc, which provides:
- Protocol conversion
- Stream multiplexing
- WebRTC/HLS/RTSP output
- Hardware acceleration support

## Troubleshooting

### Camera Not Found
- Verify the IP address is correct and reachable
- Check username/password credentials
- Ensure the camera is powered on and connected to the network
- Try using different username (some cameras use "root" or "service")

### No Streams Discovered
- Camera may use a non-standard port or protocol
- Try manual ONVIF discovery on ports 80, 8080, 2020
- Check if camera requires special authentication

### Stream Not Playing
- Ensure go2rtc is running (check logs)
- Verify browser supports WebRTC
- Check firewall settings for ports 1984, 8554, 8555
- Try using a different browser

## Advanced Configuration

### Manual Stream Addition

If automatic discovery fails, you can manually add streams by modifying the go2rtc configuration:

```yaml
streams:
  custom_camera:
    - rtsp://username:password@192.168.1.100:554/stream1
    - ffmpeg:custom_camera#video=h264#audio=aac
```

### Performance Tuning

For multiple cameras, consider:
- Using sub-streams for overview displays
- Enabling hardware acceleration
- Adjusting motion detection sensitivity
- Limiting AI detection frequency

## API Reference

### Scan Camera
```
POST /api/cameras/scan
{
  "ip": "192.168.1.100",
  "username": "admin",
  "password": "password"
}
```

### Quick Scan
```
POST /api/cameras/quick-scan
{
  "ip": "192.168.1.100",
  "username": "admin",
  "password": "password"
}
```

### Add Camera Stream
```
POST /api/cameras/add-stream
{
  "name": "Living Room",
  "stream_url": "rtsp://admin:pass@192.168.1.100:554/stream1"
}
```

## Future Enhancements

- PTZ control support
- Cloud camera integration
- NVR system detection
- Batch camera import
- Camera grouping and layouts 