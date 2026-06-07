export function looksLikeHTMLResponse(message) {
  const value = String(message || "").trim();
  return (
    value.startsWith("<!DOCTYPE html") ||
    value.startsWith("<html") ||
    value.includes("<head>") ||
    value.includes("<meta charset=") ||
    value.includes("<body") ||
    value.includes("block_url_tips") ||
    value.includes("block_message") ||
    value.includes("traceid")
  );
}

function looksLikeWrappedHTMLResponse(message) {
  const value = String(message || "").trim();
  if (!value.startsWith("{")) {
    return false;
  }

  try {
    const payload = JSON.parse(value);
    const nested = String(
      payload?.message || payload?.error || payload?.msg || "",
    ).trim();
    return looksLikeHTMLResponse(nested);
  } catch {
    return false;
  }
}

function parseJSONMessage(message) {
  const value = String(message || "").trim();
  if (!value.startsWith("{")) {
    return null;
  }

  try {
    return JSON.parse(value);
  } catch {
    return null;
  }
}

function extractNestedMessage(payload) {
  if (!payload || typeof payload !== "object") {
    return "";
  }

  const candidates = [
    payload.message,
    payload.error,
    payload.msg,
    payload.cause,
    payload.data?.message,
    payload.data?.error,
    payload.data?.msg,
  ];

  for (const candidate of candidates) {
    if (typeof candidate === "string" && candidate.trim()) {
      return candidate.trim();
    }
  }

  return "";
}

export function sanitizeErrorMessage(message) {
  const normalized = String(message || "").trim();
  if (!normalized) {
    return "";
  }

  const payload = parseJSONMessage(normalized);
  if (payload) {
    const nested = extractNestedMessage(payload);
    if (nested) {
      return sanitizeErrorMessage(nested);
    }
  }

  if (
    looksLikeHTMLResponse(normalized) ||
    looksLikeWrappedHTMLResponse(normalized)
  ) {
    return "115 接口暂时返回了异常页面，请稍后重试。";
  }

  if (normalized.toLowerCase().includes("invalid character '<'")) {
    return "115 接口暂时返回了异常页面，请稍后重试。";
  }

  return normalized;
}

export function extractErrorMessage(error) {
  if (!error) {
    return "";
  }

  if (typeof error?.cause === "string") {
    const normalizedCause = sanitizeErrorMessage(error.cause);
    if (normalizedCause) {
      return normalizedCause;
    }
  }

  if (typeof error?.data?.message === "string") {
    const normalizedDataMessage = sanitizeErrorMessage(error.data.message);
    if (normalizedDataMessage) {
      return normalizedDataMessage;
    }
  }

  if (typeof error === "string") {
    return sanitizeErrorMessage(error);
  }
  if (typeof error?.message === "string") {
    return sanitizeErrorMessage(error.message);
  }
  return sanitizeErrorMessage(String(error));
}

export function isAppNotReadyError(error) {
  return extractErrorMessage(error).toLowerCase().includes("app not ready");
}

export function isUserCancelledError(error) {
  const message = extractErrorMessage(error).toLowerCase();
  return (
    message.includes("cancelled by user") ||
    message.includes("canceled by user") ||
    message.includes("operation was canceled") ||
    message.includes("operation cancelled")
  );
}
