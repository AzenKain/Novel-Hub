import { useEffect, useRef, useState } from "react";

export const useAutoScroll = (
  contentRef: React.RefObject<HTMLElement | null>,
) => {
  const [isScrolling, setIsScrolling] = useState(false);
  const [scrollSpeed, setScrollSpeed] = useState(50);
  const animationFrameRef = useRef<number | null>(null);
  const lastTimeRef = useRef<number | null>(null);

  const toggleScroll = () => {
    setIsScrolling((prev) => !prev);
  };

  const updateSpeed = (newSpeed: number) => {
    setScrollSpeed(newSpeed);
  };

  useEffect(() => {
    if (!isScrolling || !contentRef.current) {
      if (animationFrameRef.current !== null) {
        cancelAnimationFrame(animationFrameRef.current);
        animationFrameRef.current = null;
      }
      lastTimeRef.current = null;
      return;
    }

    const scrollLoop = (timestamp: number) => {
      if (!lastTimeRef.current) {
        lastTimeRef.current = timestamp;
      }
      const deltaTime = timestamp - lastTimeRef.current;
      lastTimeRef.current = timestamp;

      if (contentRef.current) {
        const scrollAmount = (scrollSpeed * deltaTime) / 1000;
        contentRef.current.scrollTop += scrollAmount;
      }

      animationFrameRef.current = requestAnimationFrame(scrollLoop);
    };

    animationFrameRef.current = requestAnimationFrame(scrollLoop);

    return () => {
      if (animationFrameRef.current !== null) {
        cancelAnimationFrame(animationFrameRef.current);
      }
    };
  }, [isScrolling, scrollSpeed, contentRef]);

  useEffect(() => {
    const handleWheel = (e: WheelEvent) => {
      if (e.deltaY !== 0 && isScrolling) {
        setIsScrolling(false);
      }
    };

    const handleTouch = () => {
      if (isScrolling) {
        setIsScrolling(false);
      }
    };

    const node = contentRef.current;
    if (node) {
      node.addEventListener("wheel", handleWheel, { passive: true });
      node.addEventListener("touchmove", handleTouch, { passive: true });
    }

    return () => {
      if (node) {
        node.removeEventListener("wheel", handleWheel);
        node.removeEventListener("touchmove", handleTouch);
      }
    };
  }, [isScrolling, contentRef]);

  return {
    isScrolling,
    scrollSpeed,
    toggleScroll,
    updateSpeed,
    setIsScrolling,
  };
};
