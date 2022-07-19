export const CONFIG = {
    name: "vacuum",
    path: "./bin",
    url: "https://github.com/daveshanley/vacuum/releases/download/v{{version}}/{{bin_name}}_{{version}}_{{platform}}_{{arch}}.tar.gz",
};
export const ARCH_MAPPING = {
    ia32: "386",
    x64: "amd64",
    arm64: "arm64",
};
export const PLATFORM_MAPPING = {
    darwin: "darwin",
    linux: "linux",
    win32: "windows",
};