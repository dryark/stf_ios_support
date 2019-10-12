#import <Foundation/Foundation.h>
#import <AVFoundation/AVFoundation.h>
#import <CoreMediaIO/CMIOHardware.h>

int main(int argc, const char * argv[]) {
    @autoreleasepool {
        // CMI Magic to enable capture of IOS device mirroring over USB
        CMIOObjectPropertyAddress prop = {
            kCMIOHardwarePropertyAllowScreenCaptureDevices,
            kCMIOObjectPropertyScopeGlobal,
            kCMIOObjectPropertyElementMaster
        };
        UInt32 allow = 1;
        CMIOObjectSetPropertyData(kCMIOObjectSystemObject, &prop, 0, NULL, sizeof(allow), &allow );
        
        NSNotificationCenter *nCenter = [NSNotificationCenter defaultCenter];
        
        // Setup an observer for newly connected devices
        id observeConnect = [
                             nCenter addObserverForName:AVCaptureDeviceWasConnectedNotification
                             object:nil
                             queue:[NSOperationQueue mainQueue]
                             usingBlock:^(NSNotification *note)
                             {
                                 NSArray *devices = [AVCaptureDevice devices];
                                 for( AVCaptureDevice *device in devices ) {
                                     printf("Connected - ID: %s Model: %s\n", [[device uniqueID] UTF8String], [[device modelID] UTF8String]);
                                 }
                             }];
        
        // Setup an observer for disconnected devices
        id observeDisconnect = [
                                nCenter addObserverForName:AVCaptureDeviceWasDisconnectedNotification
                                object:nil
                                queue:[NSOperationQueue mainQueue]
                                usingBlock:^(NSNotification *note)
                                {
                                    NSArray *devices = [AVCaptureDevice devices];
                                    for( AVCaptureDevice *device in devices ) {
                                        printf("Disconnected - ID: %s Model: %s\n", [[device uniqueID] UTF8String], [[device modelID] UTF8String]);
                                    }
                                }];
        
        [[NSRunLoop currentRunLoop] run];
        [nCenter removeObserver:observeConnect];
        [nCenter removeObserver:observeDisconnect];
    }
    return 0;
}


