import { reatomComponent } from '@reatom/react';

interface DonutSegment {
  value: number;
  color: string;
  label: string;
}

interface DonutChartProps {
  knowCount: number;
  learningCount: number;
  dontKnowCount: number;
  size?: number;
}

/**
 * Bespoke SVG donut chart — three segments, accessible (`role="img"` + an
 * aria-label that reads the totals). No charting library, ~3KB of TSX.
 *
 * Colour scheme — derived from Tailwind tokens:
 *   - know         → brand-500 (Micocards orange)
 *   - learning     → brand-300 (softer accent)
 *   - dontKnow     → field-bg via card-border (neutral grey)
 *
 * The center number is the percentage of "Know" out of the rated total. If
 * total === 0 the chart renders a neutral ring with "0%".
 */
export const DonutChart = reatomComponent<DonutChartProps>(({
  knowCount,
  learningCount,
  dontKnowCount,
  size = 204,
}) => {
  const total = knowCount + learningCount + dontKnowCount;
  const percentage = total === 0 ? 0 : Math.round((knowCount / total) * 100);
  const segments: DonutSegment[] = [
    { value: knowCount, color: 'var(--color-brand-500)', label: 'Знаю' },
    { value: learningCount, color: 'var(--color-brand-300)', label: 'Ещё изучаю' },
    { value: dontKnowCount, color: 'var(--color-card-border)', label: 'Не знаю' },
  ];
  const radius = size / 2;
  const stroke = Math.round(size * 0.12);
  const ringRadius = radius - stroke / 2;
  const circumference = 2 * Math.PI * ringRadius;

  let cumulative = 0;
  const arcs = segments.map((seg) => {
    if (total === 0 || seg.value === 0) {
      return { ...seg, length: 0, offset: 0 };
    }
    const length = (seg.value / total) * circumference;
    const offset = -cumulative;
    cumulative += length;
    return { ...seg, length, offset };
  });

  const ariaLabel = `Знаю: ${knowCount}. Ещё изучаю: ${learningCount}. Не знаю: ${dontKnowCount}. Всего: ${total}.`;

  return (
    <svg
      role="img"
      aria-label={ariaLabel}
      width={size}
      height={size}
      viewBox={`0 0 ${size} ${size}`}
      className="block"
    >
      <circle
        cx={radius}
        cy={radius}
        r={ringRadius}
        fill="none"
        stroke="var(--color-field-bg)"
        strokeWidth={stroke}
      />
      {total > 0
        ? arcs.map((arc, idx) =>
            arc.length > 0 ? (
              <circle
                key={idx}
                cx={radius}
                cy={radius}
                r={ringRadius}
                fill="none"
                stroke={arc.color}
                strokeWidth={stroke}
                strokeDasharray={`${arc.length} ${circumference - arc.length}`}
                strokeDashoffset={arc.offset}
                transform={`rotate(-90 ${radius} ${radius})`}
                style={{ transition: 'stroke-dasharray 400ms ease-out' }}
              />
            ) : null,
          )
        : null}
      <text
        x={radius}
        y={radius}
        textAnchor="middle"
        dominantBaseline="central"
        className="fill-[var(--color-brand-600)] font-extrabold"
        style={{ fontSize: size * 0.18, fontFamily: 'Inter, sans-serif' }}
      >
        {percentage}%
      </text>
    </svg>
  );
}, 'DonutChart');
