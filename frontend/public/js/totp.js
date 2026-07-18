import {TOTP, Secret} from "https://esm.sh/otpauth@9.2.4";

export function createTOTP(secretKey) {
    return new TOTP({
        secret: Secret.fromBase32(secretKey),
        period: 30,
        digits: 6,
        algorithm: "SHA1"
    });
}

export function msUntilNext(period = 30000) {
    return period - (Date.now() % period);
}