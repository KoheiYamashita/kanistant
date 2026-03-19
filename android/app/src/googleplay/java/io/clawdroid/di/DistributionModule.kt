package io.clawdroid.di

import org.koin.core.qualifier.named
import org.koin.dsl.module

val distributionModule = module {
    single<Map<String, String>>(named("distributionEnv")) { emptyMap() }
}
