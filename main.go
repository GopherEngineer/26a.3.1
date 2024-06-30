package main

import (
	"fmt"
	"time"
)

type stageSignature func(done, input chan int) chan int

// Обработка входящего потока данных, фильтрация и передача дальше другим сдатия конвейера
func worker(done, out, input chan int, predicate func(v int) bool) {
	defer close(out)
	for {
		select {
		case <-done:
			return
		case v, isChannelOpen := <-input:
			if !isChannelOpen {
				return
			}
			if predicate(v) {
				select {
				case out <- v:
				case <-done:
					return
				}
			}
		}
	}
}

// По сути мини-фабрика для создания стадий обработки
func makeStage(predicate func(v int) bool) stageSignature {
	return func(done, input chan int) chan int {
		out := make(chan int)
		go worker(done, out, input, predicate)
		return out
	}
}

// Финальная стадия обработки данных через буферизацию
func loopBuffer(done, input chan int) chan int {
	out := make(chan int)
	// В качестве кольцевого буфера использован буферизированный канал так-как он и реализует принцип FIFO
	buffer := make(chan int, BUFFER_SIZE)

	go worker(done, buffer, input, func(v int) bool { return true })

	go func() {
		defer close(out)
		for v := range buffer {
			time.Sleep(time.Millisecond * BUFFER_DELAY)
			out <- v
		}
	}()

	return out
}

// Размер кольцевого буфера
const BUFFER_SIZE = 5

// Задержка опустошения кольцевого буфера
const BUFFER_DELAY = 100

func main() {

	// Для удобной организации стадий
	stages := make([]stageSignature, 0)
	appendToStages := func(stage stageSignature) {
		stages = append(stages, stage)
	}

	// Добавляем стадию фильтрации отрицательных значений
	appendToStages(makeStage(func(v int) bool { return v >= 0 }))
	// Добавляем стадию фильтрации значений не делящихся на 3 и равных 0
	appendToStages(makeStage(func(v int) bool { return v != 0 && v%3 == 0 }))
	// Добавляем финальную стадию обработки данных через буферизацию
	appendToStages(loopBuffer)

	// Для сигнализации всем конвейерам остановиться
	done := make(chan int)
	defer close(done)

	// Канал для получения входных данных
	source := make(chan int)

	// Объединяем стадии обработки
	pipe := stages[0](done, source)
	for _, stage := range stages[1:] {
		pipe = stage(done, pipe)
	}

	// Запускаем отдельную горутину для получние данных из консоли
	go func() {
		defer close(source)

		var input int

		for {
			if _, err := fmt.Scanln(&input); err == nil {
				// Если было получено число, то отправляем его в канал для обработки конвейером
				source <- input
			} else {
				// Завершаем цикл обработки ввода данных если получено нечисловое значение
				return
			}
		}

	}()

	// Пишем в консоль обработанные данные
	for v := range pipe {
		fmt.Printf("Обработаны данные: %d\n", v)
	}

}
