package usecase

import "fmt"

func FirstAttempt() {
	fmt.Println("🔐 First time login detected. Please scan the QR code with WhatsApp.")
	fmt.Println("📱 Steps:")
	fmt.Println("   1. Open WhatsApp on your phone")
	fmt.Println("   2. Go to Settings > Linked Devices")
	fmt.Println("   3. Tap 'Link a Device'")
	fmt.Println("   4. Scan the QR code below\n")
}
