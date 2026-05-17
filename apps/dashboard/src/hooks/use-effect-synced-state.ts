"use client";

import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type Dispatch,
  type SetStateAction,
} from "react";

export function useHydratedValue<T>(fallback: T, readValue: () => T): T {
  const [value] = useHydratedState(fallback, readValue);

  return value;
}

export function useHydratedState<T>(
  fallback: T,
  readValue: () => T,
): [T, Dispatch<SetStateAction<T>>, boolean] {
  const [state, setState] = useState({ hydrated: false, value: fallback });
  const readValueRef = useRef(readValue);

  useEffect(() => {
    readValueRef.current = readValue;
  }, [readValue]);

  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Browser-only hydration boundary; callers cannot read storage during SSR.
    setState({ hydrated: true, value: readValueRef.current() });
  }, []);

  const setValue = useCallback<Dispatch<SetStateAction<T>>>((nextValue) => {
    setState((previous) => ({
      hydrated: previous.hydrated,
      value:
        typeof nextValue === "function"
          ? (nextValue as (current: T) => T)(previous.value)
          : nextValue,
    }));
  }, []);

  return [state.value, setValue, state.hydrated];
}

export function useEffectSyncedState<T>(
  sourceValue: T,
  resetKey: string | number | boolean | null | undefined,
): [T, Dispatch<SetStateAction<T>>] {
  const [value, setValue] = useState(sourceValue);
  const previousResetKey = useRef(resetKey);

  useEffect(() => {
    if (Object.is(previousResetKey.current, resetKey)) return;

    previousResetKey.current = resetKey;
    // eslint-disable-next-line react-hooks/set-state-in-effect -- Editable local draft reset when a new external record/config is loaded.
    setValue(sourceValue);
  }, [resetKey, sourceValue]);

  return [value, setValue];
}

export function useEffectComputedState<T>(
  initialValue: T,
  computeValue: () => T | null,
): [T, Dispatch<SetStateAction<T>>] {
  const [value, setValue] = useState(initialValue);
  const computeValueRef = useRef(computeValue);

  useEffect(() => {
    computeValueRef.current = computeValue;
  }, [computeValue]);

  useEffect(() => {
    const computedValue = computeValueRef.current();
    if (computedValue === null) return;

    // eslint-disable-next-line react-hooks/set-state-in-effect -- DOM-derived measurement state must be captured after render.
    setValue(computedValue);
  }, []);

  return [value, setValue];
}
