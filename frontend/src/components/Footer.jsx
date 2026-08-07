import { Github, Twitter, Heart } from 'lucide-react';
import { motion } from 'framer-motion';

export default function Footer() {
  return (
    <motion.footer
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: 0.3, duration: 0.5 }}
      className="glass border-t border-white/20 dark:border-white/5 py-8 mt-auto relative"
    >
      {/* Gradient top border glow */}
      <div
        className="absolute top-0 left-0 right-0 h-px opacity-60"
        style={{ backgroundImage: 'linear-gradient(90deg, transparent, #0ea5e9, #8b5cf6, transparent)' }}
      />
      <div className="container mx-auto px-4 sm:px-6 lg:px-8">
        <div className="flex flex-col md:flex-row justify-between items-center space-y-4 md:space-y-0">
          <div className="text-gray-600 dark:text-gray-400 text-sm flex items-center">
            Built with
            <motion.span
              animate={{ scale: [1, 1.25, 1] }}
              transition={{ duration: 1.2, repeat: Infinity, ease: 'easeInOut' }}
              className="inline-flex mx-1"
            >
              <Heart className="w-4 h-4 text-red-500 fill-current drop-shadow-[0_0_8px_rgba(239,68,68,0.6)]" />
            </motion.span>
            by <span className="text-gradient font-semibold ml-1">LinkSnip Enterprise</span>
          </div>
          <div className="flex space-x-4">
            <motion.a
              href="#"
              whileHover={{ scale: 1.2, y: -3 }}
              whileTap={{ scale: 0.9 }}
              transition={{ type: 'spring', stiffness: 400, damping: 17 }}
              className="p-2 rounded-xl text-gray-400 hover:text-brand-600 dark:hover:text-brand-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              <Github className="w-5 h-5" />
            </motion.a>
            <motion.a
              href="#"
              whileHover={{ scale: 1.2, y: -3 }}
              whileTap={{ scale: 0.9 }}
              transition={{ type: 'spring', stiffness: 400, damping: 17 }}
              className="p-2 rounded-xl text-gray-400 hover:text-brand-600 dark:hover:text-brand-400 hover:bg-gray-100 dark:hover:bg-gray-800 transition-colors"
            >
              <Twitter className="w-5 h-5" />
            </motion.a>
          </div>
          <div className="text-gray-500 dark:text-gray-400 text-sm">
            &copy; {new Date().getFullYear()} LinkSnip. All rights reserved.
          </div>
        </div>
      </div>
    </motion.footer>
  );
}
