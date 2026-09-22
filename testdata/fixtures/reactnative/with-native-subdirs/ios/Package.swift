// swift-tools-version:5.9
// Stand-in for a Podfile-based iOS folder under an RN app; present only
// to prove the RN root's own adapter set doesn't false-positive on it.
import PackageDescription

let package = Package(name: "Pods", products: [], targets: [])
