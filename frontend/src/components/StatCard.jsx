import { clsx } from 'clsx';

export default function StatCard({ title, value, icon: Icon, trend, color = 'brand' }) {
  const colors = {
    brand: 'text-brand-600 dark:text-brand-400 bg-brand-50 dark:bg-brand-900/30',
    green: 'text-green-600 dark:text-green-400 bg-green-50 dark:bg-green-900/30',
    blue: 'text-blue-600 dark:text-blue-400 bg-blue-50 dark:bg-blue-900/30',
    purple: 'text-purple-600 dark:text-purple-400 bg-purple-50 dark:bg-purple-900/30',
    orange: 'text-orange-600 dark:text-orange-400 bg-orange-50 dark:bg-orange-900/30',
  };

  return (
    <div className="card hover:shadow-md transition-shadow">
      <div className="flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</p>
          <p className="mt-1 text-3xl font-bold text-gray-900 dark:text-white">{value}</p>
          {trend && (
            <p className="mt-1 text-xs text-green-600 dark:text-green-400 font-medium">
              {trend}
            </p>
          )}
        </div>
        <div className={clsx("p-3 rounded-lg", colors[color])}>
          <Icon className="w-6 h-6" />
        </div>
      </div>
    </div>
  );
}
