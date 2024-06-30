package main

import (
	"fmt"
	"log"
	"time"
)

type stageSignature func(done, input chan int) chan int

// Обработка входящего потока данных, фильтрация и передача дальше другим сдатия конвейера
func worker(label string, done, out, input chan int, predicate func(v int) bool) {
	defer log.Printf("Завершение стадии обработки: %s", label)
	defer close(out)
	for {
		select {
		case <-done:

			return
		case v, isChannelOpen := <-input:
			if !isChannelOpen {
				return
			}
			log.Printf("Фильтация данных стадией %s значения %d", label, v)
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
func makeStage(label string, predicate func(v int) bool) stageSignature {
	log.Printf("Создания стадии: %s", label)
	return func(done, input chan int) chan int {
		out := make(chan int)
		go worker(label, done, out, input, predicate)
		return out
	}
}

// Финальная стадия обработки данных через буферизацию
func loopBuffer(label string) stageSignature {
	return func(done, input chan int) chan int {
		out := make(chan int)
		// В качестве кольцевого буфера использован буферизированный канал так-как он и реализует принцип FIFO
		buffer := make(chan int, BUFFER_SIZE)

		go worker(label, done, buffer, input, func(v int) bool { return true })

		log.Println("Запускается отдельная горутина для работы с кольцевым буфером")
		go func() {
			defer close(out)
			for v := range buffer {
				time.Sleep(time.Millisecond * BUFFER_DELAY)
				log.Println("Запись обработанных данных в канал для итоговой печати в консоль")
				out <- v
			}
		}()

		return out
	}
}

// Размер кольцевого буфера
const BUFFER_SIZE = 5

// Задержка опустошения кольцевого буфера
const BUFFER_DELAY = 100

func main() {

	log.Println("Подготовка перед добавлением стадий обработки")
	// Для удобной организации стадий
	stages := make([]stageSignature, 0)
	appendToStages := func(label string, done func(label string) stageSignature) {
		log.Printf("Добавление стадии: %s", label)
		stages = append(stages, done(label))
	}

	// Добавляем стадию фильтрации отрицательных значений
	appendToStages("Фильтрация отрицательных значений", func(label string) stageSignature {
		return makeStage(label, func(v int) bool { return v >= 0 })
	})
	// Добавляем стадию фильтрации значений не делящихся на 3 и равных 0
	appendToStages("Фильтрация значений не делящихся на 3 и равных 0", func(label string) stageSignature {
		return makeStage(label, func(v int) bool { return v != 0 && v%3 == 0 })
	})
	// Добавляем финальную стадию обработки данных через буферизацию
	appendToStages("Кольцевой буффер", func(label string) stageSignature {
		return loopBuffer(label)
	})

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
	log.Println("Запускается отдельная горутина для получние данных из консоли")
	go func() {
		defer close(source)

		var input int

		for {
			log.Println("Ожидаем ввод данных их консоли")
			if _, err := fmt.Scanln(&input); err == nil {
				// Если было получено число, то отправляем его в канал для обработки конвейером
				log.Println("Запись полученных данных в канал для обработки конвейером")
				source <- input
			} else {
				// Завершаем цикл обработки ввода данных если получено нечисловое значение
				log.Println("Завершение цикла обработки ввода данных")
				return
			}
		}

	}()

	// Пишем в консоль обработанные данные
	for v := range pipe {
		log.Println("Запись в консоль обработанных данных")
		fmt.Printf("Обработаны данные: %d\n", v)
	}

}
