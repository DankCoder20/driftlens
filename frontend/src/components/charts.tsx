import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
const palette = [
  "#5ce4b0",
  "#7599ff",
  "#b69bff",
  "#f3be6e",
  "#f77f8e",
  "#54c8dd",
  "#8492a6",
];
const tooltip = {
  contentStyle: {
    background: "#161e2a",
    border: "1px solid #2a3547",
    borderRadius: 8,
    color: "#e8edf5",
  },
  itemStyle: { color: "#e8edf5" },
};
export function Distribution({
  data,
  donut = false,
}: {
  data: Record<string, number>;
  donut?: boolean;
}) {
  const rows = Object.entries(data)
    .map(([name, value]) => ({ name, value }))
    .sort((a, b) => b.value - a.value)
    .slice(0, 12);
  if (!rows.length)
    return <div className="chart-empty">Run a scan to collect this data.</div>;
  return (
    <div className="chart">
      <ResponsiveContainer width="100%" height={230}>
        {donut ? (
          <PieChart>
            <Pie
              data={rows}
              dataKey="value"
              nameKey="name"
              innerRadius={62}
              outerRadius={87}
              paddingAngle={3}
              stroke="none"
            >
              {rows.map((r, i) => (
                <Cell key={r.name} fill={palette[i % palette.length]} />
              ))}
            </Pie>
            <Tooltip {...tooltip} />
            <Legend iconType="circle" iconSize={7} />
          </PieChart>
        ) : (
          <BarChart
            data={rows}
            margin={{ left: -24, right: 12, top: 10, bottom: 20 }}
          >
            <CartesianGrid
              vertical={false}
              stroke="#202b3a"
              strokeDasharray="3 4"
            />
            <XAxis
              dataKey="name"
              tick={{ fill: "#8b98ac", fontSize: 10 }}
              tickLine={false}
              axisLine={false}
              interval={0}
              angle={-15}
              textAnchor="end"
            />
            <YAxis
              allowDecimals={false}
              tick={{ fill: "#8b98ac", fontSize: 11 }}
              axisLine={false}
              tickLine={false}
            />
            <Tooltip {...tooltip} cursor={{ fill: "#ffffff05" }} />
            <Bar
              dataKey="value"
              fill="#5ce4b0"
              radius={[4, 4, 0, 0]}
              maxBarSize={36}
            />
          </BarChart>
        )}
      </ResponsiveContainer>
    </div>
  );
}
export function Trend({
  rows,
  series,
}: {
  rows: Record<string, unknown>[];
  series: { key: string; name: string; color: string }[];
}) {
  if (rows.length < 2)
    return (
      <div className="chart-empty">
        <span className="pulse-dot" />
        History is accumulating.
        <small>At least two snapshots are needed to show a trend.</small>
      </div>
    );
  return (
    <div className="chart">
      <ResponsiveContainer width="100%" height={230}>
        <LineChart
          data={rows}
          margin={{ left: -24, right: 20, top: 14, bottom: 0 }}
        >
          <CartesianGrid
            vertical={false}
            stroke="#202b3a"
            strokeDasharray="3 4"
          />
          <XAxis
            dataKey="time"
            tick={{ fill: "#8b98ac", fontSize: 10 }}
            tickLine={false}
            axisLine={false}
            minTickGap={35}
          />
          <YAxis
            allowDecimals={false}
            tick={{ fill: "#8b98ac", fontSize: 11 }}
            tickLine={false}
            axisLine={false}
          />
          <Tooltip {...tooltip} />
          <Legend iconType="circle" iconSize={7} />
          {series.map((s) => (
            <Line
              key={s.key}
              type="monotone"
              dataKey={s.key}
              name={s.name}
              stroke={s.color}
              strokeWidth={2}
              dot={false}
              activeDot={{ r: 4 }}
            />
          ))}
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}
