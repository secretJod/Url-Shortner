import { clsx } from 'clsx';
import { motion, useMotionValue, useSpring, useTransform } from 'framer-motion';

export default function StatCard({ title, value, icon: Icon, trend, color = 'brand' }) {
  // 3D tilt-on-hover via pointer position
  const x = useMotionValue(0);
  const y = useMotionValue(0);
  const rotateX = useSpring(useTransform(y, [-50, 50], [8, -8]), { stiffness: 300, damping: 20 });
  const rotateY = useSpring(useTransform(x, [-50, 50], [-8, 8]), { stiffness: 300, damping: 20 });

  const colors = {
    brand: { icon: 'text-brand-600 dark:text-brand-400', bg: 'bg-brand-50 dark:bg-brand-900/30', glow: 'rgba(14,165,233,0.35)' },
    green: { icon: 'text-green-600 dark:text-green-400', bg: 'bg-green-50 dark:bg-green-900/30', glow: 'rgba(16,185,129,0.35)' },
    blue: { icon: 'text-blue-600 dark:text-blue-400', bg: 'bg-blue-50 dark:bg-blue-900/30', glow: 'rgba(59,130,246,0.35)' },
    purple: { icon: 'text-purple-600 dark:text-purple-400', bg: 'bg-purple-50 dark:bg-purple-900/30', glow: 'rgba(139,92,246,0.35)' },
    orange: { icon: 'text-orange-600 dark:text-orange-400', bg: 'bg-orange-50 dark:bg-orange-900/30', glow: 'rgba(249,115,22,0.35)' },
  };

  const c = colors[color] || colors.brand;

  const handleMouse = (e) => {
    const rect = e.currentTarget.getBoundingClientRect();
    x.set(e.clientX - rect.left - rect.width / 2);
    y.set(e.clientY - rect.top - rect.height / 2);
  };

  const reset = () => {
    x.set(0);
    y.set(0);
  };

  return (
    <motion.div
      onMouseMove={handleMouse}
      onMouseLeave={reset}
      style={{ rotateX, rotateY, transformPerspective: 800 }}
      whileHover={{ scale: 1.03 }}
      transition={{ type: 'spring', stiffness: 300, damping: 20 }}
      className="card relative overflow-hidden group"
    >
      {/* Shimmer sweep on hover */}
      <div className="absolute inset-0 opacity-0 group-hover:opacity-100 transition-opacity duration-500 pointer-events-none">
        <div
          className="absolute -inset-1 shimmer-surface"
          style={{ background: `linear-gradient(110deg, transparent 30%, ${c.glow} 50%, transparent 70%)` }}
        />
      </div>

      <div className="relative flex items-center justify-between">
        <div>
          <p className="text-sm font-medium text-gray-500 dark:text-gray-400">{title}</p>
          <motion.p
            initial={{ opacity: 0, y: 8 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
            className="mt-1 text-3xl font-bold text-gray-900 dark:text-white"
          >
            {value}
          </motion.p>
          {trend && (
            <p className="mt-1 text-xs text-green-600 dark:text-green-400 font-medium">
              {trend}
            </p>
          )}
        </div>
        <motion.div
          whileHover={{ rotate: 10, scale: 1.1 }}
          transition={{ type: 'spring', stiffness: 400, damping: 15 }}
          className={clsx("relative p-3 rounded-xl", c.bg)}
          style={{ boxShadow: `0 0 20px ${c.glow}` }}
        >
          <Icon className={clsx("w-6 h-6", c.icon)} />
        </motion.div>
      </div>
    </motion.div>
  );
}
