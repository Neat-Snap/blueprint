export type PasswordPolicy = {
  minLength: number;
  maxLength: number;
  requireUpper: boolean;
  requireLower: boolean;
  requireNumber: boolean;
  requireSymbol: boolean;
};

function getBool(envVal: string | undefined, def: boolean): boolean {
  if (!envVal) return def;
  switch (envVal.toLowerCase()) {
    case "1":
    case "true":
    case "t":
    case "yes":
    case "y":
    case "on":
      return true;
    case "0":
    case "false":
    case "f":
    case "no":
    case "n":
    case "off":
      return false;
    default:
      return def;
  }
}

export function getPasswordPolicy(): PasswordPolicy {
  const def: PasswordPolicy = {
    minLength: 8,
    maxLength: 128,
    requireUpper: true,
    requireLower: true,
    requireNumber: true,
    requireSymbol: true,
  };

  const min = Number(process.env.NEXT_PUBLIC_PASSWORD_MIN_LENGTH || def.minLength);
  const max = Number(process.env.NEXT_PUBLIC_PASSWORD_MAX_LENGTH || def.maxLength);
  return {
    minLength: Number.isFinite(min) && min > 0 ? min : def.minLength,
    maxLength: Number.isFinite(max) && max > 0 ? max : def.maxLength,
    requireUpper: getBool(process.env.NEXT_PUBLIC_PASSWORD_REQUIRE_UPPER, def.requireUpper),
    requireLower: getBool(process.env.NEXT_PUBLIC_PASSWORD_REQUIRE_LOWER, def.requireLower),
    requireNumber: getBool(process.env.NEXT_PUBLIC_PASSWORD_REQUIRE_NUMBER, def.requireNumber),
    requireSymbol: getBool(process.env.NEXT_PUBLIC_PASSWORD_REQUIRE_SYMBOL, def.requireSymbol),
  };
}

const symbolRe = /[!@#$%^&*()_+\-=[\]{}|;':",./<>?~]/;

export function validateEmail(email: string): string | null {
  const e = (email || "").trim().toLowerCase();
  if (!e) return "Email is required";

  const re = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  if (!re.test(e)) return "Invalid email format";
  return null;
}

export function validatePassword(pw: string, policy: PasswordPolicy = getPasswordPolicy()): string | null {
  if (!pw) return "Password is required";
  if (pw.length < policy.minLength) return `Password must be at least ${policy.minLength} characters`;
  if (policy.maxLength && pw.length > policy.maxLength) return `Password must be at most ${policy.maxLength} characters`;
  if (policy.requireUpper && !/[A-Z]/.test(pw)) return "Password must contain an uppercase letter";
  if (policy.requireLower && !/[a-z]/.test(pw)) return "Password must contain a lowercase letter";
  if (policy.requireNumber && !/[0-9]/.test(pw)) return "Password must contain a number";
  if (policy.requireSymbol && !symbolRe.test(pw)) return "Password must contain a symbol";
  return null;
}
