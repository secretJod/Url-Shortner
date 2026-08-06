import { clsx } from 'clsx';

export default function Spinner({ size = 'md', className = '' }) {
  const sizes = {
    sm: 'w-4 h-4',
    md: 'w-8 h-8',
    lg: 'w-12 h-12'
  };

  return (
    <div className={clsx("flex justify-center items-center p-4", className)}>
      <div
        className={clsx(
          "animate-spin rounded-full border-2 border-gray-200 dark:border-gray-700 border-t-brand-600 dark:border-t-brand-400",
          sizes[size]
        )}
      />
    </div>
  );
}
