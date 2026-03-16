"use client";

interface ConfidenceGaugeProps {
  confidence: number;
  className?: string;
}

function getConfidenceColor(value: number): string {
  if (value <= 50) return "bg-red-500";
  if (value <= 75) return "bg-yellow-500";
  return "bg-green-500";
}

function getConfidenceLabel(value: number): string {
  if (value <= 50) return "LOW";
  if (value <= 75) return "";
  return "";
}

function getConfidenceTextColor(value: number): string {
  if (value <= 50) return "text-red-400";
  if (value <= 75) return "text-yellow-400";
  return "text-green-400";
}

export function ConfidenceGauge({ confidence, className = "" }: ConfidenceGaugeProps) {
  const color = getConfidenceColor(confidence);
  const label = getConfidenceLabel(confidence);
  const textColor = getConfidenceTextColor(confidence);

  return (
    <div
      data-testid="confidence-gauge"
      className={`flex items-center gap-2 ${className}`}
    >
      <div className="flex items-center gap-1.5">
        <div className="w-16 h-1.5 rounded-full bg-[var(--bg-tertiary)] overflow-hidden">
          <div
            className={`h-full rounded-full transition-all ${color}`}
            style={{ width: `${confidence}%` }}
          />
        </div>
        <span className={`text-xs font-mono font-medium ${textColor}`}>
          {confidence}%
        </span>
      </div>
      {label && (
        <span className={`text-xs font-semibold ${textColor}`}>
          {label}
        </span>
      )}
    </div>
  );
}
